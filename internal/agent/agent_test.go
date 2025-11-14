package agent //nolint: testpackage //let me be

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	models "github.com/ttl256/metrics/internal/model"
	"resty.dev/v3"
)

func TestSendMetrics(t *testing.T) {
	var count int
	counterHandler := func(_ http.ResponseWriter, _ *http.Request) {
		count++
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/", counterHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	v := 13.37
	client := resty.New().SetBaseURL(server.URL)
	m := Metrics{
		metrics: models.Metrics{
			ID:    "test",
			MType: models.Gauge,
			Delta: nil,
			Value: &v,
			Hash:  "",
		},
		client: client,
	}
	ctx := context.Background()
	err := m.Send(ctx, "update/")

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
