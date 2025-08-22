//go:build prommetrics

package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/pkg/agentclient"
)

// mountCombinedMetricsRoutes registers /metrics/all that concatenates nosd and agent metrics.
func mountCombinedMetricsRoutes(cfg config.Config, r chi.Router) {
	r.Get("/metrics/all", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		// nosd metrics
		rr := httptest.NewRecorder()
		promhttp.HandlerFor(prom.DefaultGatherer, promhttp.HandlerOpts{}).ServeHTTP(rr, req)
		_, _ = io.Copy(w, rr.Body)
		// separator
		_, _ = w.Write([]byte("# agent metrics\n"))
		// agent metrics via unix socket HTTP
		client := agentclient.New(cfg.AgentSocket())
		ctx, cancel := context.WithTimeout(req.Context(), 500*time.Millisecond)
		defer cancel()
		httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/metrics", nil)
		res, err := client.HTTP.Do(httpReq)
		if err != nil {
			_, _ = w.Write([]byte("# agent metrics unavailable: " + err.Error() + "\n"))
			return
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 {
			_, _ = w.Write([]byte("# agent metrics unavailable: status " + fmt.Sprint(res.StatusCode) + "\n"))
			return
		}
		_, _ = io.Copy(w, res.Body)
	})
}
