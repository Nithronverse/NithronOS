//go:build !prommetrics

package server

import (
	"nithronos/backend/nosd/internal/config"

	"github.com/go-chi/chi/v5"
)

func mountCombinedMetricsRoutes(cfg config.Config, r chi.Router) {
	// no-op when prommetrics tag not enabled
}
