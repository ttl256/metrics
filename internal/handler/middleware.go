package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	xerrors "github.com/pkg/errors"
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

func gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

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

		h.ServeHTTP(ow, r)
	})
}
