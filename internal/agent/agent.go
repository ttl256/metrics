package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"runtime"
	"time"

	"github.com/cenkalti/backoff/v5"
	xerrors "github.com/pkg/errors"
	"resty.dev/v3"

	models "github.com/ttl256/metrics/internal/model"
)

type Metrics struct {
	metrics models.Metrics
	client  *resty.Client
}

func (m *Metrics) Send(ctx context.Context, path string) error {
	const maxRetryTime = 1 * time.Minute
	_, err := backoff.Retry(
		ctx, func() (bool, error) {
			return true, m.send(ctx, path)
		},
		backoff.WithMaxElapsedTime(maxRetryTime),
	)
	return xerrors.WithStack(err)
}

func (m *Metrics) send(ctx context.Context, path string) error {
	log := slog.Default().With(
		slog.String("uri", path),
		slog.Any("metrics", m.metrics),
	)
	log.DebugContext(ctx, "sending request")
	body, err := json.Marshal(m.metrics)
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err = gzWriter.Write(body)
	if err != nil {
		return fmt.Errorf("compressing: %w", err)
	}
	err = gzWriter.Close()
	if err != nil {
		return fmt.Errorf("compressing: %w", err)
	}
	resp, err := m.client.R().
		SetContentType("application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetContext(ctx).
		SetBody(buf.Bytes()).
		Post(path)
	if err != nil {
		log.ErrorContext(ctx, "getting response", "error", xerrors.WithStack(err))
		return xerrors.WithStack(err)
	}
	if !resp.IsSuccess() {
		log.ErrorContext(
			ctx,
			"unsuccessful response",
			slog.Int("code", resp.StatusCode()),
			slog.Any("error", resp.Err),
		)
		return fmt.Errorf("unexpected status code %d", resp.StatusCode())
	}
	log.DebugContext(ctx, "response ok")
	return nil
}

type Agent struct {
	url            string
	metrics        []models.Metrics
	pollInterval   time.Duration
	reportInterval time.Duration
	counter        int64
}

func NewAgent(
	endpoint string,
	pollInterval time.Duration,
	reportInterval time.Duration,
) *Agent {
	return &Agent{
		url:            endpoint,
		metrics:        nil,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		counter:        0,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	log := slog.Default()
	log.InfoContext(
		ctx,
		"starting agent",
		slog.Duration("poll_interval", a.pollInterval),
		slog.Duration("report_interval", a.reportInterval),
	)
	client := resty.New().SetBaseURL(a.url)
	for {
		select {
		case <-pollTicker.C:
			a.metrics = a.BuildRuntimeMetrics()
			a.metrics = append(a.metrics, models.Metrics{
				ID:    `PollCount`,
				MType: models.Counter,
				Delta: &a.counter,
				Value: nil,
				Hash:  "",
			})
		case <-reportTicker.C:
			for _, m := range a.metrics {
				mm := &Metrics{
					metrics: m,
					client:  client,
				}
				err := mm.Send(ctx, "update/")
				if err != nil {
					return err
				}
			}
			a.counter = 0
		case <-ctx.Done():
			return xerrors.WithStack(ctx.Err())
		}
	}
}

func (a *Agent) BuildRuntimeMetrics() []models.Metrics {
	m := &runtime.MemStats{} //nolint: exhaustruct //let me be
	runtime.ReadMemStats(m)
	log := slog.Default()
	log.Debug("build runtime metrics")
	a.counter++
	return []models.Metrics{
		NewGaugeMetric("Alloc", float64(m.Alloc)),
		NewGaugeMetric("BuckHashSys", float64(m.BuckHashSys)),
		NewGaugeMetric("Frees", float64(m.Frees)),
		NewGaugeMetric("Frees", float64(m.Frees)),
		NewGaugeMetric("GCCPUFraction", float64(m.GCCPUFraction)),
		NewGaugeMetric("GCSys", float64(m.GCSys)),
		NewGaugeMetric("HeapAlloc", float64(m.HeapAlloc)),
		NewGaugeMetric("HeapIdle", float64(m.HeapIdle)),
		NewGaugeMetric("HeapInuse", float64(m.HeapInuse)),
		NewGaugeMetric("HeapObjects", float64(m.HeapObjects)),
		NewGaugeMetric("HeapReleased", float64(m.HeapReleased)),
		NewGaugeMetric("HeapSys", float64(m.HeapSys)),
		NewGaugeMetric("LastGC", float64(m.LastGC)),
		NewGaugeMetric("Lookups", float64(m.Lookups)),
		NewGaugeMetric("MCacheInuse", float64(m.MCacheInuse)),
		NewGaugeMetric("MCacheSys", float64(m.MCacheSys)),
		NewGaugeMetric("MSpanInuse", float64(m.MSpanInuse)),
		NewGaugeMetric("MSpanSys", float64(m.MSpanSys)),
		NewGaugeMetric("Mallocs", float64(m.Mallocs)),
		NewGaugeMetric("NextGC", float64(m.NextGC)),
		NewGaugeMetric("NumForcedGC", float64(m.NumForcedGC)),
		NewGaugeMetric("NumGC", float64(m.NumGC)),
		NewGaugeMetric("OtherSys", float64(m.OtherSys)),
		NewGaugeMetric("PauseTotalNs", float64(m.PauseTotalNs)),
		NewGaugeMetric("StackInuse", float64(m.StackInuse)),
		NewGaugeMetric("StackSys", float64(m.StackSys)),
		NewGaugeMetric("Sys", float64(m.Sys)),
		NewGaugeMetric("TotalAlloc", float64(m.TotalAlloc)),
		NewGaugeMetric("RandomValue", rand.Float64()), //nolint: gosec //let me be
	}
}

func NewGaugeMetric(name string, value float64) models.Metrics {
	return models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Delta: nil,
		Value: &value,
		Hash:  "",
	}
}
