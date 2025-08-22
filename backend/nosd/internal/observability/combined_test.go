package observability

import (
    "context"
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    prom "github.com/prometheus/client_golang/prometheus"
)

type fakeAgent struct{ url string }

func (f fakeAgent) FetchMetrics(ctx context.Context) ([]byte, error) {
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
    res, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()
    return io.ReadAll(res.Body)
}

func TestCombinedMetricsHandler(t *testing.T) {
    // Stub agent server
    agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain; version=0.0.4")
        _, _ = w.Write([]byte("nithronos_agent_build_info 1\n"))
    }))
    defer agentSrv.Close()

    // Test registry with one counter
    reg := prom.NewRegistry()
    c := prom.NewCounter(prom.CounterOpts{Name: "nithronos_dummy_total"})
    reg.MustRegister(c)
    c.Inc()

    // Handler under test
    h := NewCombinedMetricsHandler(reg, fakeAgent{url: agentSrv.URL})

    // Exercise handler
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/metrics/all", nil)
    h.ServeHTTP(rr, req)

    body := rr.Body.String()
    if !strings.Contains(body, "nithronos_dummy_total") {
        t.Fatalf("expected nosd metric in body, got: %s", body)
    }
    if !strings.Contains(body, "nithronos_agent_build_info") {
        t.Fatalf("expected agent metric in body, got: %s", body)
    }
}


