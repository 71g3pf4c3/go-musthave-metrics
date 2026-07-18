package agent_test

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/compress"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/sign"
)

func newAgent(srv *httptest.Server, extra ...func(*config.AgentConfig)) *agent.Agent {
	cfg := config.AgentConfig{
		Address:        srv.URL,
		PollInterval:   1,
		ReportInterval: 1,
		RateLimit:      1,
	}
	for _, fn := range extra {
		fn(&cfg)
	}
	a, err := agent.New(cfg)
	if err != nil {
		panic(err)
	}
	return a
}

func TestCollectAndReport_BatchSentToUpdates(t *testing.T) {
	var mu sync.Mutex
	calledUpdates := false
	receivedCount := 0

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/updates" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var batch []models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		calledUpdates = true
		receivedCount = len(batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	a := newAgent(srv)
	a.Collect()
	a.CollectExtra()

	batch := a.BuildBatch()
	if len(batch) == 0 {
		t.Fatal("expected non-empty batch")
	}

	a.SendBatch(batch)

	mu.Lock()
	defer mu.Unlock()
	if !calledUpdates {
		t.Fatal("expected /updates to be called")
	}
	if receivedCount == 0 {
		t.Fatal("expected non-empty batch on server")
	}
}

func TestReport_FallbackToSingleMetricAPI(t *testing.T) {
	var mu sync.Mutex
	fallbackCalls := 0

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/updates":
			w.WriteHeader(http.StatusInternalServerError)
		case "/update":
			var m models.Metrics
			if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mu.Lock()
			fallbackCalls++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	a := newAgent(srv)
	a.Collect()
	batch := a.BuildBatch()
	a.SendBatch(batch)

	mu.Lock()
	defer mu.Unlock()
	if fallbackCalls == 0 {
		t.Fatal("expected fallback /update calls")
	}
}

func TestReport_GzipEncodingHeader(t *testing.T) {
	var mu sync.Mutex
	allGzip := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Header.Get("Content-Encoding") != "gzip" {
			allGzip = false
		}
		mu.Unlock()

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("body is not valid gzip: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gz.Close()

		if r.URL.Path == "/updates" {
			var batch []models.Metrics
			if err := json.NewDecoder(gz).Decode(&batch); err != nil {
				t.Errorf("failed to decode gzip JSON batch: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		} else {
			var m models.Metrics
			if err := json.NewDecoder(gz).Decode(&m); err != nil {
				t.Errorf("failed to decode gzip JSON metric: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := newAgent(srv)
	a.Collect()
	a.SendBatch(a.BuildBatch())

	mu.Lock()
	defer mu.Unlock()
	if !allGzip {
		t.Error("expected all requests to have Content-Encoding: gzip")
	}
}

func TestReport_SetsHashHeaderWhenKeyProvided(t *testing.T) {
	gotHash := ""

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/updates" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotHash = r.Header.Get(sign.HeaderHashSHA256)
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	a := newAgent(srv, func(cfg *config.AgentConfig) { cfg.Key = "test-key" })
	a.Collect()
	a.SendBatch(a.BuildBatch())

	if gotHash == "" {
		t.Fatal("expected HashSHA256 header to be set")
	}
}

func TestCollectExtra_SetsMemAndCPU(t *testing.T) {
	a := newAgent(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	a.CollectExtra()
	batch := a.BuildBatch()

	gauges := map[string]bool{}
	for _, m := range batch {
		if m.MType == models.Gauge {
			gauges[m.ID] = true
		}
	}

	if !gauges["TotalMemory"] {
		t.Error("expected TotalMemory gauge")
	}
	if !gauges["FreeMemory"] {
		t.Error("expected FreeMemory gauge")
	}

	found := false
	for k := range gauges {
		if len(k) > 14 && k[:14] == "CPUutilization" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one CPUutilizationN gauge")
	}
}
