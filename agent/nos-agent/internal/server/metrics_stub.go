//go:build !prommetrics

package server

import "net/http"

func initMetrics() {}
func metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
}
func incBtrfsStatus(kind string)                      {}
func observeBtrfsStatus(kind string, seconds float64) {}
