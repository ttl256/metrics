package agent //nolint: testpackage //let me be

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	models "github.com/ttl256/metrics/internal/model"
)

func TestSendMetrics(t *testing.T) {
	var count int
	counterHandler := func(_ http.ResponseWriter, _ *http.Request) {
		count++
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /updates/", counterHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	v := 13.37
	agent := NewAgent(server.URL, time.Duration(0), time.Duration(0), nil)
	agent.AddMetric(models.Metrics{
		ID:    "test",
		MType: models.Gauge,
		Delta: nil,
		Value: &v,
		Hash:  "",
	})
	ctx := context.Background()
	err := agent.Send(ctx, "updates/")

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
