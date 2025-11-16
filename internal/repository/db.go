package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cenkalti/backoff/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/ttl256/metrics/database"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
	migrations "github.com/ttl256/metrics/internal/sql"
)

type DBStorage struct {
	db      *pgxpool.Pool
	queries *database.Queries
	logger  *slog.Logger
}

func NewDBStorage(ctx context.Context, dsn string) (*DBStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	q := database.New(pool)
	return &DBStorage{
		db:      pool,
		queries: q,
		logger:  slog.Default(),
	}, nil
}

func (m *DBStorage) Close() {
	m.db.Close()
}

func (m *DBStorage) Save(ctx context.Context, metric models.Metrics) (err error) {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", errRollback))
			}
		}
	}()
	qtx := m.queries.WithTx(tx)
	err = qtx.CreateMetric(ctx, database.CreateMetricParams(toRepo(metric)))
	if err != nil {
		return fmt.Errorf("inserting metric: %w", err)
	}
	return nil
}

func (m *DBStorage) SaveMany(ctx context.Context, metrics []models.Metrics) (err error) {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", errRollback))
			}
		}
	}()
	qtx := m.queries.WithTx(tx)
	for _, metric := range metrics {
		err = qtx.CreateMetric(ctx, database.CreateMetricParams(toRepo(metric)))
		if err != nil {
			return fmt.Errorf("inserting metric: %w", err)
		}
	}
	return nil
}

func (m *DBStorage) Get(ctx context.Context, id string) (models.Metrics, error) {
	repoMetric, err := m.queries.GetMetricByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			m.logger.DebugContext(ctx, "getting metric", slog.String("id", id), slog.Any("error", err))
			return models.Metrics{}, service.ErrMetricsNotFound
		}
		m.logger.ErrorContext(ctx, "getting metrics", slog.String("id", id), slog.Any("error", err))
		return models.Metrics{}, fmt.Errorf("getting metric %q: %w", id, err)
	}
	return toDomain(repoMetric), nil
}

func (m *DBStorage) GetAll(ctx context.Context) ([]models.Metrics, error) {
	repoMetrics, err := m.queries.GetMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting metrics: %w", err)
	}
	metrics := make([]models.Metrics, 0, len(repoMetrics))
	for _, i := range repoMetrics {
		metrics = append(metrics, toDomain(i))
	}
	return metrics, nil
}

func (m *DBStorage) RepoPing(ctx context.Context) error {
	var attempt int
	_, err := backoff.Retry(ctx, func() (bool, error) {
		attempt++
		m.logger.InfoContext(ctx, "connecting to db", slog.Int("attempt", attempt))
		err := m.db.Ping(ctx)
		if err != nil {
			m.logger.Error("failed connecting to db", slog.Any("error", err))
			return true, fmt.Errorf("ping db: %w", err)
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	m.logger.InfoContext(ctx, "connected to db", slog.Int("attempt", attempt))
	return nil
}

func (m *DBStorage) Migrate() error {
	iofsDriver, err := iofs.New(migrations.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("creating iofs driver: %w", err)
	}
	dbDriver, err := postgres.WithInstance(stdlib.OpenDBFromPool(m.db), new(postgres.Config))
	if err != nil {
		return fmt.Errorf("creating database driver with instance: %w", err)
	}
	defer dbDriver.Close()

	mig, err := migrate.NewWithInstance("iofs", iofsDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("instantiating migration: %w", err)
	}
	if err = mig.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			m.logger.Error("performing UP database migration", slog.Any("error", err))
			return fmt.Errorf("performing UP database migration: %w", err)
		}
		m.logger.Info("performing migration: no change")
		return nil
	}
	m.logger.Info("migration is complete")
	return nil
}
