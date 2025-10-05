package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/repository"
	"github.com/ttl256/metrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	app := App{
		service: service.NewService(repository.NewMemStorage()),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.healthHandler)
	mux.Handle("POST /update/{type}", http.NotFoundHandler())
	mux.HandleFunc("POST /update/{type}/{name}/{value}", app.updateHandler)
	mux.HandleFunc("GET /{id}", app.getHandler)
	mux.HandleFunc("GET /all", app.getAllHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("serving HTTP: %w", err)
	}
	return nil
}

var (
	ErrInvalidMetricsType  = errors.New("invalid metrics type")
	ErrInvalidMetricsValue = errors.New("invalid metrics value")
	ErrInvalidMetricsName  = errors.New("invalid metrics name")
)

type App struct {
	service *service.Service
}

func (a App) updateHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := newMetrics(r.PathValue("type"), r.PathValue("name"), r.PathValue("value"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = a.service.Save(metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a App) getHandler(w http.ResponseWriter, r *http.Request) {
	metricsID := r.PathValue("id")
	if metricsID == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}
	metrics, err := a.service.Get(metricsID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func (a App) getAllHandler(w http.ResponseWriter, _ *http.Request) {
	metrics, err := a.service.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func (a App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	type resp struct {
		Status string `json:"status"`
	}
	data, err := json.Marshal(resp{Status: `OK`})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func newMetrics(_type, name, value string) (models.Metrics, error) {
	if name == "" {
		return models.Metrics{}, fmt.Errorf("%w: %q", ErrInvalidMetricsName, name)
	}

	switch _type {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return models.Metrics{}, fmt.Errorf("%w: %q", ErrInvalidMetricsValue, value)
		}
		return models.Metrics{
			ID:    name,
			MType: _type,
			Delta: nil,
			Value: &v,
			Hash:  "",
		}, nil
	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return models.Metrics{}, fmt.Errorf("%w: %q", ErrInvalidMetricsValue, value)
		}
		return models.Metrics{
			ID:    name,
			MType: _type,
			Delta: &v,
			Value: nil,
			Hash:  "",
		}, nil
	default:
		return models.Metrics{}, fmt.Errorf("%w: %q", ErrInvalidMetricsType, _type)
	}
}
