package observability

import (
	"context"
	"net/http"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// AgentMetricsClient defines the minimal interface required to fetch agent metrics.
type AgentMetricsClient interface {
	FetchMetrics(ctx context.Context) ([]byte, error)
}

// NewCombinedMetricsHandler returns an http.Handler that writes Prometheus text metrics
// for the provided nosd gatherer followed by agent metrics if available.
func NewCombinedMetricsHandler(g prom.Gatherer, agentClient AgentMetricsClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Write nosd metrics using promhttp against the provided gatherer
		promhttp.HandlerFor(g, promhttp.HandlerOpts{}).ServeHTTP(w, r)

		// Attempt to fetch agent metrics with a short timeout
		if agentClient == nil {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		data, err := agentClient.FetchMetrics(ctx)
		if err != nil {
			_, _ = w.Write([]byte("# agent metrics unavailable: " + err.Error() + "\n"))
			return
		}
		if len(data) > 0 {
			_, _ = w.Write([]byte("# --- agent metrics below ---\n"))
			_, _ = w.Write(data)
		}
	})
}
