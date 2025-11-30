package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"math/rand/v2"
	"runtime"
	"slices"
	"time"

	"github.com/cenkalti/backoff/v5"
	xerrors "github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"golang.org/x/sync/errgroup"
	"resty.dev/v3"

	"github.com/ttl256/metrics/internal/common"
	models "github.com/ttl256/metrics/internal/model"
)

type Agent struct {
	url            string
	metrics        map[string]models.Metrics
	pollInterval   time.Duration
	reportInterval time.Duration
	counter        int64
	rateLimit      chan struct{}
	logger         *slog.Logger
	client         *resty.Client
	key            []byte
}

func NewAgent(
	endpoint string,
	pollInterval time.Duration,
	reportInterval time.Duration,
	rateLimit int,
	key []byte,
) *Agent {
	return &Agent{
		url:            endpoint,
		metrics:        make(map[string]models.Metrics),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		counter:        0,
		rateLimit:      make(chan struct{}, max(1, rateLimit)),
		logger:         slog.Default(),
		client:         resty.New().SetBaseURL(endpoint),
		key:            key,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	// pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	a.logger.InfoContext(
		ctx,
		"starting agent",
		slog.String("poll_interval", a.pollInterval.String()),
		slog.String("report_interval", a.reportInterval.String()),
	)
	metricsCh := make(chan []models.Metrics)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return a.BuildRuntimeMetrics(ctx, metricsCh)
	})
	g.Go(func() error {
		return a.BuildSysMetrics(ctx, metricsCh)
	})
	g.Go(func() error {
		for {
			select {
			case metrics := <-metricsCh:
				for _, i := range metrics {
					a.metrics[i.ID] = i
				}
				a.counter++
				a.metrics[`PollCount`] = models.Metrics{
					ID:    `PollCount`,
					MType: models.Counter,
					Delta: &a.counter,
					Value: nil,
					Hash:  "",
				}
			case <-reportTicker.C:
				err := a.Send(ctx, "updates/")
				if err != nil {
					return err
				}
				a.counter = 0
			case <-ctx.Done():
				return xerrors.WithStack(ctx.Err())
			}
		}
	})
	err := g.Wait()
	close(metricsCh)
	if err != nil {
		return xerrors.WithStack(err)
	}
	return nil
}

func (a *Agent) AddMetric(m models.Metrics) {
	a.metrics[m.ID] = m
}

func (a *Agent) Send(ctx context.Context, path string) error {
	a.rateLimit <- struct{}{}
	defer func() { <-a.rateLimit }()
	const maxRetryTime = 1 * time.Minute
	_, err := backoff.Retry(
		ctx, func() (bool, error) {
			return true, a.send(ctx, path)
		},
		backoff.WithMaxElapsedTime(maxRetryTime),
	)
	return xerrors.WithStack(err)
}

func (a *Agent) send(ctx context.Context, path string) error {
	log := a.logger.With(
		slog.String("uri", path),
		slog.Any("metrics", a.metrics),
	)
	log.DebugContext(ctx, "sending request")
	body, err := json.Marshal(slices.Collect(maps.Values(a.metrics)))
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
	req := a.client.R()
	if len(a.key) != 0 {
		var computedHash string
		computedHash, err = common.Hash(body, a.key)
		if err != nil {
			return fmt.Errorf("computing hash: %w", err)
		}
		req.SetHeader(common.HashHeader, computedHash)
	}
	resp, err := req.
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

func (a *Agent) BuildRuntimeMetrics(ctx context.Context, out chan<- []models.Metrics) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m := &runtime.MemStats{} //nolint: exhaustruct //let me be
			runtime.ReadMemStats(m)
			a.logger.DebugContext(ctx, "build runtime metrics")
			metrics := []models.Metrics{
				NewGaugeMetric("Alloc", float64(m.Alloc)),
				NewGaugeMetric("BuckHashSys", float64(m.BuckHashSys)),
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
			select {
			case out <- metrics:
			case <-ctx.Done():
			}
		case <-ctx.Done():
			return xerrors.WithStack(ctx.Err())
		}
	}
}

func (a *Agent) BuildSysMetrics(ctx context.Context, out chan<- []models.Metrics) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			memStat, err := mem.VirtualMemory()
			if err != nil {
				return fmt.Errorf("getting memory metrics: %w", err)
			}
			cpuStat, err := cpu.Percent(0, true)
			if err != nil {
				return fmt.Errorf("getting cpu metrics: %w", err)
			}
			a.logger.DebugContext(ctx, "build system metrics")
			metrics := []models.Metrics{
				NewGaugeMetric("TotalMemory", float64(memStat.Total)),
				NewGaugeMetric("FreeMemory", float64(memStat.Free)),
			}
			for i, stat := range cpuStat {
				metrics = append(metrics, NewGaugeMetric(
					fmt.Sprintf("CPUutilization%d", i),
					stat,
				))
			}
			select {
			case out <- metrics:
			case <-ctx.Done():
			}
		case <-ctx.Done():
			return xerrors.WithStack(ctx.Err())
		}
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
