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
	"github.com/ttl256/metrics/internal/migrations"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
)

type DBStorage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	return &DBStorage{
		db:     db,
		logger: slog.Default(),
	}, nil
}

func (m *DBStorage) Save(ctx context.Context, metric models.Metrics) error {
	r, err := m.db.ExecContext(
		ctx,
		"insert into metric (id, type, delta, value, hash) values ($1, $2, $3, $4, $5)"+
			" on conflict (id) do update set delta = EXCLUDED.delta, value=EXCLUDED.value",
		metric.ID, metric.MType, metric.Delta, metric.Value, metric.Hash,
	)
	if err != nil {
		m.logger.ErrorContext(ctx, "inserting metric", slog.Any("metric", metric), slog.Any("error", err))
		return fmt.Errorf("inserting metric: %w", err)
	}
	n, err := r.RowsAffected()
	if err != nil {
		m.logger.ErrorContext(ctx, "inserting metric", slog.Any("metric", metric), slog.Any("error", err))
		return fmt.Errorf("inserting metric: %w", err)
	}
	m.logger.DebugContext(ctx, "rows affected", slog.Int64("value", n))
	return nil
}

func (m *DBStorage) Get(ctx context.Context, id string) (models.Metrics, error) {
	row := m.db.QueryRowContext(ctx, "select id, type, delta, value, hash from metric where id = $1", id)
	var metric models.Metrics
	err := row.Scan(
		&metric.ID,
		&metric.MType,
		&metric.Delta,
		&metric.Value,
		&metric.Hash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			m.logger.DebugContext(ctx, "getting metric", slog.String("id", id), slog.Any("error", err))
			return metric, service.ErrMetricsNotFound
		}
		m.logger.ErrorContext(ctx, "getting metrics", slog.String("id", id), slog.Any("error", err))
		return metric, fmt.Errorf("getting metric %q: %w", id, err)
	}
	return metric, nil
}

func (m *DBStorage) GetAll(ctx context.Context) ([]models.Metrics, error) {
	rows, err := m.db.QueryContext(ctx, "select id, type, delta, value, hash from metric")
	if err != nil {
		return nil, fmt.Errorf("getting metrics: %w", err)
	}
	defer rows.Close()
	var metrics []models.Metrics
	for rows.Next() {
		var m models.Metrics
		err = rows.Scan(
			&m.ID,
			&m.MType,
			&m.Delta,
			&m.Value,
			&m.Hash,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		metrics = append(metrics, m)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning rows: %w", err)
	}
	return metrics, nil
}

func (m *DBStorage) RepoPing(ctx context.Context) error {
	var attempt int
	_, err := backoff.Retry(ctx, func() (bool, error) {
		attempt++
		m.logger.InfoContext(ctx, "connecting to db", slog.Int("attempt", attempt))
		err := m.db.PingContext(ctx)
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

	dbDriver, err := postgres.WithInstance(m.db, new(postgres.Config))
	if err != nil {
		return fmt.Errorf("creating database driver with instance: %w", err)
	}

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
	}
	m.logger.Info("migration is complete")

	return nil
}
