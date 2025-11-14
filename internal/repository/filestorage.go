package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"time"

	xerrors "github.com/pkg/errors"

	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
)

type FileStorage struct {
	metrics       map[string]models.Metrics
	file          *os.File
	storeInterval time.Duration
}

func NewFileStorage(path string, storeInterval time.Duration, restore bool) (*FileStorage, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening storage file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("getting stat on file: %w", err)
	}
	metricsM := make(map[string]models.Metrics)
	if restore && info.Size() > 0 {
		dec := json.NewDecoder(f)
		var metricsS []models.Metrics
		err = dec.Decode(&metricsS)
		if err != nil {
			return nil, fmt.Errorf("decoding storage file contents: %w", err)
		}
		for _, i := range metricsS {
			metricsM[i.ID] = i
		}
	}
	return &FileStorage{
		metrics:       metricsM,
		file:          f,
		storeInterval: storeInterval,
	}, nil
}

func (s *FileStorage) Save(metric models.Metrics) error {
	s.metrics[metric.ID] = metric
	if err := s.file.Truncate(0); err != nil {
		return xerrors.WithStack(err)
	}
	if _, err := s.file.Seek(0, 0); err != nil {
		return xerrors.WithStack(err)
	}
	enc := json.NewEncoder(s.file)
	enc.SetIndent("", "  ")
	metrics, _ := s.GetAll()
	if err := enc.Encode(metrics); err != nil {
		return xerrors.WithStack(err)
	}
	return nil
}

func (s *FileStorage) Get(id string) (models.Metrics, error) {
	v, ok := s.metrics[id]
	if !ok {
		return models.Metrics{}, service.ErrMetricsNotFound
	}
	return v, nil
}

func (s *FileStorage) GetAll() ([]models.Metrics, error) {
	return slices.Collect(maps.Values(s.metrics)), nil
}

func (s *FileStorage) RepoPing(_ context.Context) error {
	return nil
}

func (s *FileStorage) Close() error {
	return xerrors.WithStack(s.file.Close())
}
