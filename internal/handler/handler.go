package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/service"
)

var (
	ErrInvalidMetricsType  = errors.New("invalid metrics type")
	ErrInvalidMetricsValue = errors.New("invalid metrics value")
	ErrInvalidMetricsName  = errors.New("invalid metrics name")
)

type MetricsService interface {
	Save(context.Context, models.Metrics) error
	SaveMany(context.Context, []models.Metrics) error
	Get(context.Context, string) (models.Metrics, error)
	GetAll(context.Context) ([]models.Metrics, error)
	RepoPing(context.Context) error
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
	const compressionLevel = 5
	r := chi.NewRouter()
	log := slog.Default()
	r.Use(httplog.RequestLogger(log, nil))
	r.Use(middleware.Compress(compressionLevel, "text/html", "application/json"))
	r.Use(gzipMiddleware)
	r.Get("/healthz", h.HealthHandler)
	r.Get("/value/{type}/{name}", h.GetHandler)
	r.Get("/", h.GetAllHandler)
	r.Get("/all", h.GetAllHandler)
	r.Post("/update/{type}/{name}/{value}", h.UpdateHandler)
	r.Post("/update/", h.UpdateHandlerJSON)
	r.Post("/updates/", h.UpdateManyHandlerJSON)
	r.Post("/value/", h.GetHandlerJSON)
	r.Get("/ping", h.Ping)

	return r
}

func (h *HTTPHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := newMetrics(r.PathValue("type"), r.PathValue("name"), r.PathValue("value"))
	if err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err = h.svc.Save(r.Context(), metrics); err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) UpdateHandlerJSON(w http.ResponseWriter, r *http.Request) {
	var metrics models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	switch metrics.MType {
	case models.Gauge:
		if metrics.Value == nil {
			slog.Default().Error("", slog.Any("error", errors.New("value is not set for gauge metric")))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	case models.Counter:
		if metrics.Delta == nil {
			slog.Default().Error("", slog.Any("error", errors.New("delta is not set for counter metric")))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	if err := h.svc.Save(r.Context(), metrics); err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) UpdateManyHandlerJSON(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				slog.Default().Error("", slog.Any("error", errors.New("value is not set for gauge metric")))
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		case models.Counter:
			if metric.Delta == nil {
				slog.Default().Error("", slog.Any("error", errors.New("delta is not set for counter metric")))
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}
	}
	if err := h.svc.SaveMany(r.Context(), metrics); err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	metricsID := r.PathValue("name")
	if metricsID == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}
	metrics, err := h.svc.Get(r.Context(), metricsID)
	if err != nil {
		if errors.Is(err, service.ErrMetricsNotFound) {
			slog.Default().Error("", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		slog.Default().Error("", slog.Any("error", err))
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

func (h *HTTPHandler) GetHandlerJSON(w http.ResponseWriter, r *http.Request) {
	var metricsReq models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metricsReq); err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	metrics, err := h.svc.Get(r.Context(), metricsReq.ID)
	if err != nil {
		if errors.Is(err, service.ErrMetricsNotFound) {
			slog.Default().Error("", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *HTTPHandler) GetAllHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetAll(r.Context())
	if err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	var buf bytes.Buffer
	buf.WriteString("<html>")
	buf.Write(data)
	buf.WriteString("</html>")
	_, _ = w.Write(buf.Bytes())
}

func (h *HTTPHandler) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	data, err := json.Marshal(HealthResponse{Status: `OK`})
	if err != nil {
		slog.Default().Error("", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *HTTPHandler) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.svc.RepoPing(r.Context())
	if err != nil {
		http.Error(w, "repository is unavailable", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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
