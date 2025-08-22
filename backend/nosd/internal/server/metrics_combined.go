package server

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	expfmt "github.com/prometheus/common/expfmt"
)

// AgentMetricsClient fetches Prometheus text-format metrics from the agent.
type AgentMetricsClient interface {
	FetchMetrics(ctx context.Context) ([]byte, error)
}

// NewCombinedMetricsHandler serves nosd metrics and appends agent metrics.
// Content-Type follows Prometheus text exposition format.
func NewCombinedMetricsHandler(g prometheus.Gatherer, agent AgentMetricsClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prometheus text format content type
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Gather nosd metrics
		mfs, err := g.Gather()
		if err != nil {
			http.Error(w, "gather metrics failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, mf := range mfs {
			if _, err := expfmt.MetricFamilyToText(w, mf); err != nil {
				return
			}
		}

		// Separator
		_, _ = w.Write([]byte("\n# --- agent metrics below ---\n"))

		// Append agent metrics (best-effort)
		if agent == nil {
			_, _ = w.Write([]byte("# agent metrics unavailable: no client\n"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()

		b, err := agent.FetchMetrics(ctx)
		if err != nil {
			_, _ = w.Write([]byte("# agent metrics unavailable: " + err.Error() + "\n"))
			return
		}
		_, _ = w.Write(b)
	})
}
