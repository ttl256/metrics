package handler //nolint: testpackage //let me be

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	models "github.com/ttl256/metrics/internal/model"
	"github.com/ttl256/metrics/internal/repository"
	"github.com/ttl256/metrics/internal/service"
)

func TestAppHealthHandler(t *testing.T) {
	a := NewApp(nil)
	srv := httptest.NewServer(a.GetRouter())
	defer srv.Close()

	client := resty.New()
	resp, err := client.R().Get(srv.URL + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	expectedBody, err := json.Marshal(HealthResponse{Status: `OK`})
	require.NoError(t, err)
	assert.JSONEq(t, string(expectedBody), string(resp.Body()))
}

func TestAppUpdateHandler(t *testing.T) {
	a := NewApp(service.NewService(repository.NewMemStorage()))
	srv := httptest.NewServer(a.GetRouter())
	defer srv.Close()

	client := resty.New()

	t.Run("successful update", func(t *testing.T) {
		var (
			_type = models.Gauge
			name  = "test1"
			value = 13.37
		)
		resp, err := client.R().Post(srv.URL + "/update/" + fmt.Sprintf("%s/%s/%f", _type, name, value))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		resp, err = client.R().Get(srv.URL + "/value/" + fmt.Sprintf("%s/%s", _type, name))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		got, err := strconv.ParseFloat(string(resp.Body()), 64)
		require.NoError(t, err)
		assert.InEpsilon(t, value, got, 1e-9)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := client.R().Post(srv.URL + "/update/" + "counter")
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
	})
}
