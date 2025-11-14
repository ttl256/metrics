package service

import (
	"errors"
	"fmt"

	xerrors "github.com/pkg/errors"

	models "github.com/ttl256/metrics/internal/model"
)

type MetricsRepository interface {
	Save(models.Metrics) error
	Get(string) (models.Metrics, error)
	GetAll() ([]models.Metrics, error)
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

func (s *Service) Save(metric models.Metrics) error {
	m, err := s.repo.Get(metric.ID)
	if err != nil {
		if errors.Is(err, ErrMetricsNotFound) {
			return xerrors.WithStack(s.repo.Save(metric))
		}
		return xerrors.WithStack(err)
	}
	switch metric.MType {
	case models.Gauge:
		return xerrors.WithStack(s.repo.Save(metric))
	case models.Counter:
		return xerrors.WithStack(s.repo.Save(NewCounterMetric(metric.ID, *m.Delta+*metric.Delta)))
	default:
		return errors.Join(errUpdateMetrics, fmt.Errorf("unknown metrics type %q", metric.MType))
	}
}

func (s *Service) Get(id string) (models.Metrics, error) {
	metrics, err := s.repo.Get(id)
	if err != nil {
		return models.Metrics{}, xerrors.WithStack(err)
	}
	return metrics, nil
}

func (s *Service) GetAll() ([]models.Metrics, error) {
	metrics, err := s.repo.GetAll()
	if err != nil {
		return nil, xerrors.WithStack(err)
	}
	return metrics, nil
}
