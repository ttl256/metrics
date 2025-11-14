package repository

import (
	"context"
	"maps"
	"slices"

	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
)

type MemStorage struct {
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (m *MemStorage) Save(_ context.Context, metric models.Metrics) error {
	m.metrics[metric.ID] = metric
	return nil
}

func (m *MemStorage) Get(_ context.Context, id string) (models.Metrics, error) {
	v, ok := m.metrics[id]
	if !ok {
		return models.Metrics{}, service.ErrMetricsNotFound
	}
	return v, nil
}

func (m *MemStorage) GetAll(_ context.Context) ([]models.Metrics, error) {
	return slices.Collect(maps.Values(m.metrics)), nil
}

func (m *MemStorage) RepoPing(_ context.Context) error {
	return nil
}
