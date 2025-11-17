package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/ttl256/metrics/database"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
	migrations "github.com/ttl256/metrics/internal/sql"
)

type DBOptions struct {
	DSN                   string
	ApplicationName       string
	ConnectTimeout        time.Duration
	StatementTimeout      time.Duration
	LockTimeout           time.Duration
	IdleInTxTimeout       time.Duration
	PoolMaxConns          int32
	PoolMinConns          int32
	MaxConnLifetime       time.Duration
	MaxConnLifetimeJitter time.Duration
	MaxConnIdleTime       time.Duration
	HealthCheckPeriod     time.Duration
}

type DBStorage struct {
	db              *pgxpool.Pool
	queries         *database.Queries
	logger          *slog.Logger
	errorClassifier *PostgresErrorClassifier
}

func NewDBStorage(ctx context.Context, opts DBOptions) (*DBStorage, error) {
	if opts.DSN == "" {
		return nil, errors.New("DSN must be provided")
	}
	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}
	if opts.PoolMaxConns > 0 {
		cfg.MaxConns = opts.PoolMaxConns
	}
	if opts.PoolMinConns > 0 {
		cfg.MinConns = opts.PoolMinConns
	}
	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.MaxConnLifetimeJitter > 0 {
		cfg.MaxConnLifetimeJitter = opts.MaxConnLifetimeJitter
	}
	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	if opts.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = opts.HealthCheckPeriod
	}
	if opts.ConnectTimeout > 0 {
		cfg.ConnConfig.Config.ConnectTimeout = opts.ConnectTimeout
	}

	runtimeParams := cfg.ConnConfig.Config.RuntimeParams
	if runtimeParams == nil {
		runtimeParams = make(map[string]string)
	}
	if opts.ApplicationName != "" {
		runtimeParams["application_name"] = opts.ApplicationName
	}
	if opts.StatementTimeout > 0 {
		runtimeParams["statement_timeout"] = opts.StatementTimeout.String()
	}
	if opts.LockTimeout > 0 {
		runtimeParams["lock_timeout"] = opts.LockTimeout.String()
	}
	if opts.IdleInTxTimeout > 0 {
		runtimeParams["idle_in_transaction_session_timeout"] = opts.IdleInTxTimeout.String()
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	q := database.New(pool)
	return &DBStorage{
		db:              pool,
		queries:         q,
		logger:          slog.Default(),
		errorClassifier: NewPostgresErrorClassifier(),
	}, nil
}

func (m *DBStorage) Close() {
	m.db.Close()
}

func (m *DBStorage) Save(ctx context.Context, metric models.Metrics) error {
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		err := m.save(ctx, metric)
		if err != nil {
			if m.errorClassifier.Classify(err) == Permanent {
				return struct{}{}, backoff.Permanent(err)
			}
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	if err != nil {
		return fmt.Errorf("saving metric %q: %w", metric.ID, err)
	}
	return nil
}

func (m *DBStorage) save(ctx context.Context, metric models.Metrics) (err error) {
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

func (m *DBStorage) SaveMany(ctx context.Context, metrics []models.Metrics) error {
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		err := m.saveMany(ctx, metrics)
		if err != nil {
			if m.errorClassifier.Classify(err) == Permanent {
				return struct{}{}, backoff.Permanent(err)
			}
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	if err != nil {
		return fmt.Errorf("saving metrics %v: %w", metrics, err)
	}
	return nil
}

func (m *DBStorage) saveMany(ctx context.Context, metrics []models.Metrics) (err error) {
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
	metrics, err := backoff.Retry(ctx, func() (models.Metrics, error) {
		metricsS, err := m.get(ctx, id)
		if err != nil {
			if m.errorClassifier.Classify(err) == Permanent {
				return models.Metrics{}, backoff.Permanent(err)
			}
			return metricsS, err
		}
		return metricsS, nil
	})
	if err != nil {
		return models.Metrics{}, fmt.Errorf("getting metrics %q: %w", id, err)
	}
	return metrics, nil
}

func (m *DBStorage) get(ctx context.Context, id string) (models.Metrics, error) {
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
	metrics, err := backoff.Retry(ctx, func() ([]models.Metrics, error) {
		metrics, err := m.getAll(ctx)
		if err != nil {
			if m.errorClassifier.Classify(err) == Permanent {
				return nil, backoff.Permanent(err)
			}
			return nil, err
		}
		return metrics, nil
	})
	if err != nil {
		return nil, fmt.Errorf("getting metrics: %w", err)
	}
	return metrics, nil
}

func (m *DBStorage) getAll(ctx context.Context) ([]models.Metrics, error) {
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
	_ = pgerrcode.ConnectionException
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
