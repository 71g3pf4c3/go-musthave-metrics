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

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	mu.Lock()
	defer mu.Unlock()

	if !calledUpdates {
		t.Fatal("expected /updates to be called")
	}
	if receivedCount == 0 {
		t.Fatal("expected non-empty batch")
	}
}

func TestReport_FallbackToSingleMetricAPI(t *testing.T) {
	var mu sync.Mutex
	fallbackCalls := 0

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/updates":
			w.WriteHeader(http.StatusInternalServerError)
			return
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
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})))
	defer srv.Close()

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

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

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

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

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10, Key: "test-key"})
	a.Collect()
	a.Report()

	if gotHash == "" {
		t.Fatal("expected HashSHA256 header to be set")
	}
}
