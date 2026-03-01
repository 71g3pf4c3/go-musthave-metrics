package agent_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
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
	var received []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received = append(received, r.URL.Path)
		mu.Unlock()
		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := agent.New(srv.URL, agent.AgentConfig{PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	for _, name := range expectedGaugeNames {
		found := false
		for _, path := range received {
			if strings.Contains(path, "/update/gauge/"+name+"/") {
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
	for _, path := range received {
		if strings.Contains(path, "/update/counter/PollCount/") {
			pollSent = true
			break
		}
	}
	if !pollSent {
		t.Error("counter PollCount not sent")
	}
}

func TestCollectAndReport_ContentTypeIsTextPlain(t *testing.T) {
	var mu sync.Mutex
	badRequests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Header.Get("Content-Type") != "text/plain" {
			badRequests++
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := agent.New(srv.URL, agent.AgentConfig{PollInterval: 2, ReportInterval: 10})
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

	a := agent.New(srv.URL, agent.AgentConfig{PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	if wrongMethod > 0 {
		t.Errorf("got %d requests with wrong HTTP method", wrongMethod)
	}
}

func TestReport_PollCountIncrementsEachCollect(t *testing.T) {
	var mu sync.Mutex
	var received []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received = append(received, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := agent.New(srv.URL, agent.AgentConfig{PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Collect()
	a.Collect()
	a.Report()

	for _, path := range received {
		if strings.HasPrefix(path, "/update/counter/PollCount/") {
			suffix := strings.TrimPrefix(path, "/update/counter/PollCount/")
			if suffix != "3" {
				t.Errorf("expected PollCount=3 after 3 collects, got %q", suffix)
			}
			return
		}
	}
	t.Error("PollCount not found in sent requests")
}

func TestReport_URLFormat(t *testing.T) {
	var mu sync.Mutex
	var paths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := agent.New(srv.URL, agent.AgentConfig{PollInterval: 2, ReportInterval: 10})
	a.Collect()
	a.Report()

	for _, path := range paths {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) != 4 {
			t.Errorf("unexpected URL format: %q (want /update/<type>/<name>/<value>)", path)
		}
		if parts[0] != "update" {
			t.Errorf("expected path to start with /update, got %q", path)
		}
		metricType := parts[1]
		if metricType != "gauge" && metricType != "counter" {
			t.Errorf("unexpected metric type %q in path %q", metricType, path)
		}
	}
}
