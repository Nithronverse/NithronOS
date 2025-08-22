//go:build prommetrics

package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	prom "github.com/prometheus/client_golang/prometheus"

	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/observability"
	"nithronos/backend/nosd/pkg/agentclient"
)

// mountCombinedMetricsRoutes registers /metrics/all that concatenates nosd and agent metrics.
func mountCombinedMetricsRoutes(cfg config.Config, r chi.Router) {
	type agentFetcher struct{ client *agentclient.Client }
	func (a agentFetcher) FetchMetrics(ctx context.Context) ([]byte, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/metrics", nil)
		res, err := a.client.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 {
			return nil, io.ErrUnexpectedEOF
		}
		return io.ReadAll(res.Body)
	}

	h := observability.NewCombinedMetricsHandler(prom.DefaultGatherer, agentFetcher{client: agentclient.New(cfg.AgentSocket())})
	r.Get("/metrics/all", h.ServeHTTP)
}
