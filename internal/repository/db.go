package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/cenkalti/backoff/v5"
	models "github.com/ttl256/metrics/internal/model"
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

func (m *DBStorage) Save(_ models.Metrics) error {
	return nil
}

func (m *DBStorage) Get(_ string) (models.Metrics, error) {
	return models.Metrics{}, nil
}

func (m *DBStorage) GetAll() ([]models.Metrics, error) {
	return nil, nil
}

func (m *DBStorage) RepoPing(ctx context.Context) error {
	var attempt int
	_, err := backoff.Retry(ctx, func() (bool, error) {
		attempt++
		m.logger.Info("connecting to db", slog.Int("attempt", attempt))
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
	return nil
}
