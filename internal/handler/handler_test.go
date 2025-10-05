package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApp_HealthHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	a := NewApp(nil)
	w := httptest.NewRecorder()
	a.HealthHandler(w, request)
	res := w.Result()
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
