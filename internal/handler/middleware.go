package handler

import (
	"net/http"
	"time"

	xerrors "github.com/pkg/errors"
	"github.com/ttl256/metrics/internal/logger"
	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter

	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	if err != nil {
		return 0, xerrors.WithStack(err)
	}
	r.responseData.size += size
	return size, nil
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func (h *HTTPHandler) WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.RequestURI
		method := r.Method

		respData := &responseData{
			status: 0,
			size:   0,
		}
		loggerW := &loggingResponseWriter{
			ResponseWriter: w,
			responseData:   respData,
		}

		next.ServeHTTP(loggerW, r)

		logger.Log.Info(
			"http",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Duration("duration", time.Since(start)),
			zap.Int("status", respData.status),
			zap.Int("size", respData.size),
		)
	})
}
