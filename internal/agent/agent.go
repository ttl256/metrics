package agent

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	neturl "net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	xerrors "github.com/pkg/errors"

	models "github.com/ttl256/metrics/internal/model"
)

type Metrics struct {
	metrics models.Metrics
	client  *http.Client
}

func (m *Metrics) Send(ctx context.Context, url string) error {
	path, err := neturl.JoinPath(url, "update", metricsToPath(m.metrics))
	if err != nil {
		return xerrors.WithStack(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	if err != nil {
		return xerrors.WithStack(err)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return xerrors.WithStack(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
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
	client := &http.Client{}
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
				err := mm.Send(ctx, a.url)
				if err != nil {
					return err
				}
				a.counter = 0
			}
		case <-ctx.Done():
			return xerrors.WithStack(ctx.Err())
		}
	}
}

func (a *Agent) BuildRuntimeMetrics() []models.Metrics {
	m := &runtime.MemStats{} //nolint: exhaustruct //let me be
	runtime.ReadMemStats(m)
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

func metricsToPath(metrics models.Metrics) string {
	switch metrics.MType {
	case models.Gauge:
		return strings.Join([]string{metrics.MType, metrics.ID, strconv.FormatFloat(*metrics.Value, 'f', -1, 64)}, "/")
	case models.Counter:
		return strings.Join([]string{metrics.MType, metrics.ID, strconv.FormatInt(*metrics.Delta, 10)}, "/")
	default:
		return ""
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
