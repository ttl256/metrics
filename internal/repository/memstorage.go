package repository

import (
	"errors"
	"maps"
	"slices"

	models "github.com/ttl256/metrics/internal/model"
)

var (
	ErrMetricsNotFound = errors.New("requested metrics not found")
)

type MemStorage struct {
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (m MemStorage) Save(metric models.Metrics) error {
	m.metrics[metric.ID] = metric
	return nil
}

func (m MemStorage) Get(id string) (models.Metrics, error) {
	v, ok := m.metrics[id]
	if !ok {
		return models.Metrics{}, ErrMetricsNotFound
	}
	return v, nil
}

func (m MemStorage) GetAll() ([]models.Metrics, error) {
	return slices.Collect(maps.Values(m.metrics)), nil
}
