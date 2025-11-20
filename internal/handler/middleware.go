package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"

	xerrors "github.com/pkg/errors"
	"github.com/ttl256/metrics/internal/common"
)

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, xerrors.WithStack(err)
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c *compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p) //nolint: wrapcheck //fine
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return xerrors.WithStack(err)
	}
	return xerrors.WithStack(c.zr.Close())
}

func (m *HTTPHandler) gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendsGzip := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(w, r)
	})
}

func (m *HTTPHandler) validateHash(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := m.logger.With(slog.String("middleware", "validateHash"))
		if len(m.key) == 0 {
			h.ServeHTTP(w, r)
			return
		}
		hashHeader := r.Header.Get(common.HashHeader)
		if hashHeader == "" {
			// log.DebugContext(r.Context(), "HashSHA256 header missing")
			// http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			h.ServeHTTP(w, r)
			return
		}
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.WarnContext(r.Context(), "reading request body", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		computedHash, err := common.Hash(reqBody, m.key)
		if err != nil {
			log.WarnContext(r.Context(), "computing hash", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if hashHeader != computedHash {
			log.WarnContext(r.Context(), "hashes do not match")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		log.DebugContext(r.Context(), "hash verified")
		h.ServeHTTP(w, r)
	})
}
