package service

import (
	"context"
	"errors"
	"fmt"

	xerrors "github.com/pkg/errors"

	models "github.com/ttl256/metrics/internal/model"
)

type MetricsRepository interface {
	Save(context.Context, models.Metrics) error
	Get(context.Context, string) (models.Metrics, error)
	GetAll(context.Context) ([]models.Metrics, error)
	RepoPing(context.Context) error
}

var (
	errUpdateMetrics   = errors.New("cannot update metrics")
	ErrMetricsNotFound = errors.New("requested metrics not found")
)

func NewCounterMetric(name string, value int64) models.Metrics {
	return models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &value,
		Value: nil,
		Hash:  "",
	}
}

type Service struct {
	repo MetricsRepository
}

func NewService(repo MetricsRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Save(ctx context.Context, metric models.Metrics) error {
	m, err := s.repo.Get(ctx, metric.ID)
	if err != nil {
		if errors.Is(err, ErrMetricsNotFound) {
			return xerrors.WithStack(s.repo.Save(ctx, metric))
		}
		return xerrors.WithStack(err)
	}
	switch metric.MType {
	case models.Gauge:
		return xerrors.WithStack(s.repo.Save(ctx, metric))
	case models.Counter:
		return xerrors.WithStack(s.repo.Save(ctx, NewCounterMetric(metric.ID, *m.Delta+*metric.Delta)))
	default:
		return errors.Join(errUpdateMetrics, fmt.Errorf("unknown metrics type %q", metric.MType))
	}
}

func (s *Service) Get(ctx context.Context, id string) (models.Metrics, error) {
	metrics, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.Metrics{}, xerrors.WithStack(err)
	}
	return metrics, nil
}

func (s *Service) GetAll(ctx context.Context) ([]models.Metrics, error) {
	metrics, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, xerrors.WithStack(err)
	}
	return metrics, nil
}

func (s *Service) RepoPing(ctx context.Context) error {
	return xerrors.WithStack(s.repo.RepoPing(ctx))
}
