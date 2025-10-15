package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ttl256/metrics/internal/config"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
)

var (
	ErrInvalidMetricsType  = errors.New("invalid metrics type")
	ErrInvalidMetricsValue = errors.New("invalid metrics value")
	ErrInvalidMetricsName  = errors.New("invalid metrics name")
)

type App struct {
	config  *config.Application
	service *service.Service
}

func NewApp(cfg *config.Application, s *service.Service) *App {
	return &App{
		config:  cfg,
		service: s,
	}
}

func (a App) Run() error {
	server := &http.Server{
		Addr:         a.config.Address,
		Handler:      a.GetRouter(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("serving HTTP: %w", err)
	}
	return nil
}

func (a App) GetRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/healthz", a.HealthHandler)
	r.Get("/value/{type}/{name}", a.GetHandler)
	r.Get("/all", a.GetAllHandler)
	r.Post("/update/{type}/{name}/{value}", a.UpdateHandler)

	return r
}

func (a App) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := newMetrics(r.PathValue("type"), r.PathValue("name"), r.PathValue("value"))
	if err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err = a.service.Save(metrics); err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (a App) GetHandler(w http.ResponseWriter, r *http.Request) {
	metricsID := r.PathValue("name")
	if metricsID == "" {
		// TODO: log error
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}
	metrics, err := a.service.Get(metricsID)
	if err != nil {
		if errors.Is(err, service.ErrMetricsNotFound) {
			// TODO: log error
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var value string
	switch metrics.MType {
	case models.Gauge:
		value = strconv.FormatFloat(*metrics.Value, 'f', -1, 64)
	case models.Counter:
		value = strconv.FormatInt(*metrics.Delta, 10)
	}
	_, _ = w.Write([]byte(value))
}

func (a App) GetAllHandler(w http.ResponseWriter, _ *http.Request) {
	metrics, err := a.service.GetAll()
	if err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func (a App) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	data, err := json.Marshal(HealthResponse{Status: `OK`})
	if err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
