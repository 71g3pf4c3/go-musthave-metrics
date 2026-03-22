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
)

var expectedGaugeNames = []string{
	"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
	"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
	"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
	"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
	"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
	"Sys", "TotalAlloc", "RandomValue",
}

func TestCollectAndReport_AllMetricsSent(t *testing.T) {
	var mu sync.Mutex
	var received []models.Metrics

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, m)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	for _, name := range expectedGaugeNames {
		found := false
		for _, m := range received {
			if m.ID == name && m.MType == models.Gauge && m.Value != nil {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("gauge metric %q not sent", name)
		}
	}

	// PollCount counter must be sent
	pollSent := false
	for _, m := range received {
		if m.ID == "PollCount" && m.MType == models.Counter && m.Delta != nil {
			pollSent = true
			break
		}
	}
	if !pollSent {
		t.Error("counter PollCount not sent")
	}
}

func TestCollectAndReport_ContentTypeIsJSON(t *testing.T) {
	var mu sync.Mutex
	badRequests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Header.Get("Content-Type") != "application/json" {
			badRequests++
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	if badRequests > 0 {
		t.Errorf("got %d requests with wrong Content-Type", badRequests)
	}
}

func TestCollectAndReport_MethodIsPost(t *testing.T) {
	var mu sync.Mutex
	wrongMethod := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Method != http.MethodPost {
			wrongMethod++
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	if wrongMethod > 0 {
		t.Errorf("got %d requests with wrong HTTP method", wrongMethod)
	}
}

func TestReport_PollCountIncrementsEachCollect(t *testing.T) {
	var mu sync.Mutex
	var received []models.Metrics

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, m)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Collect()
	a.Collect()
	a.Report()

	for _, m := range received {
		if m.ID == "PollCount" && m.MType == models.Counter {
			if m.Delta == nil || *m.Delta != 3 {
				t.Errorf("expected PollCount=3 after 3 collects, got %v", m.Delta)
			}
			return
		}
	}
	t.Error("PollCount not found in sent requests")
}

func TestReport_SendError(t *testing.T) {
	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	addr := srv.URL
	srv.Close()

	a := agent.New(config.AgentConfig{Address: addr, PollInterval: 1, ReportInterval: 1})
	a.Collect()
	a.Report() // must not panic
}

func TestReport_JSONFormat(t *testing.T) {
	var mu sync.Mutex
	var received []models.Metrics

	srv := httptest.NewServer(compress.CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update" {
			t.Errorf("expected path /update, got %q", r.URL.Path)
		}
		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("failed to decode JSON body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, m)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	a := agent.New(config.AgentConfig{Address: srv.URL, PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	for _, m := range received {
		if m.ID == "" {
			t.Error("metric ID is empty")
		}
		if m.MType != models.Gauge && m.MType != models.Counter {
			t.Errorf("unexpected metric type %q", m.MType)
		}
		if m.MType == models.Gauge && m.Value == nil {
			t.Errorf("gauge metric %q has nil Value", m.ID)
		}
		if m.MType == models.Counter && m.Delta == nil {
			t.Errorf("counter metric %q has nil Delta", m.ID)
		}
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

		// verify body is valid gzip
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("body is not valid gzip: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gz.Close()

		var m models.Metrics
		if err := json.NewDecoder(gz).Decode(&m); err != nil {
			t.Errorf("failed to decode gzip JSON body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
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
