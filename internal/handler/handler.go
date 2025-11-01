package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
)

var (
	ErrInvalidMetricsType  = errors.New("invalid metrics type")
	ErrInvalidMetricsValue = errors.New("invalid metrics value")
	ErrInvalidMetricsName  = errors.New("invalid metrics name")
)

type MetricsService interface {
	Save(models.Metrics) error
	Get(string) (models.Metrics, error)
	GetAll() ([]models.Metrics, error)
}

type HTTPHandler struct {
	svc MetricsService
}

func NewHTTPHandler(svc MetricsService) *HTTPHandler {
	return &HTTPHandler{
		svc: svc,
	}
}

func (h *HTTPHandler) Routes() *chi.Mux {
	r := chi.NewRouter()
	// r.Get("/healthz", h.HealthHandler)
	r.Method(http.MethodGet, "/healthz", h.WithLogging(http.HandlerFunc(h.HealthHandler)))
	// r.Get("/value/{type}/{name}", h.GetHandler)
	r.Method(http.MethodGet, "/value/{type}/{name}", h.WithLogging(http.HandlerFunc(h.GetHandler)))
	// r.Get("/all", h.GetAllHandler)
	r.Method(http.MethodGet, "/all", h.WithLogging(http.HandlerFunc(h.GetAllHandler)))
	// r.Post("/update/{type}/{name}/{value}", h.UpdateHandler)
	r.Method(http.MethodPost, "/update/{type}/{name}/{value}", h.WithLogging(http.HandlerFunc(h.UpdateHandler)))

	return r
}

func (h *HTTPHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := newMetrics(r.PathValue("type"), r.PathValue("name"), r.PathValue("value"))
	if err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err = h.svc.Save(metrics); err != nil {
		// TODO: log error
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	metricsID := r.PathValue("name")
	if metricsID == "" {
		// TODO: log error
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}
	metrics, err := h.svc.Get(metricsID)
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
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(value))
}

func (h *HTTPHandler) GetAllHandler(w http.ResponseWriter, _ *http.Request) {
	metrics, err := h.svc.GetAll()
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
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *HTTPHandler) HealthHandler(w http.ResponseWriter, _ *http.Request) {
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
