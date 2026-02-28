package agent

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
)

type Agent struct {
	client     *http.Client
	serverAddr string
	gauges     map[string]float64
	pollCount  int64
	mu         sync.Mutex
}

func New(serverAddr string) *Agent {
	return &Agent{
		client:     &http.Client{},
		serverAddr: serverAddr,
		gauges:     make(map[string]float64),
	}
}

func (a *Agent) Collect() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	a.mu.Lock()
	defer a.mu.Unlock()

	a.gauges["Alloc"] = float64(ms.Alloc)
	a.gauges["BuckHashSys"] = float64(ms.BuckHashSys)
	a.gauges["Frees"] = float64(ms.Frees)
	a.gauges["GCCPUFraction"] = ms.GCCPUFraction
	a.gauges["GCSys"] = float64(ms.GCSys)
	a.gauges["HeapAlloc"] = float64(ms.HeapAlloc)
	a.gauges["HeapIdle"] = float64(ms.HeapIdle)
	a.gauges["HeapInuse"] = float64(ms.HeapInuse)
	a.gauges["HeapObjects"] = float64(ms.HeapObjects)
	a.gauges["HeapReleased"] = float64(ms.HeapReleased)
	a.gauges["HeapSys"] = float64(ms.HeapSys)
	a.gauges["LastGC"] = float64(ms.LastGC)
	a.gauges["Lookups"] = float64(ms.Lookups)
	a.gauges["MCacheInuse"] = float64(ms.MCacheInuse)
	a.gauges["MCacheSys"] = float64(ms.MCacheSys)
	a.gauges["MSpanInuse"] = float64(ms.MSpanInuse)
	a.gauges["MSpanSys"] = float64(ms.MSpanSys)
	a.gauges["Mallocs"] = float64(ms.Mallocs)
	a.gauges["NextGC"] = float64(ms.NextGC)
	a.gauges["NumForcedGC"] = float64(ms.NumForcedGC)
	a.gauges["NumGC"] = float64(ms.NumGC)
	a.gauges["OtherSys"] = float64(ms.OtherSys)
	a.gauges["PauseTotalNs"] = float64(ms.PauseTotalNs)
	a.gauges["StackInuse"] = float64(ms.StackInuse)
	a.gauges["StackSys"] = float64(ms.StackSys)
	a.gauges["Sys"] = float64(ms.Sys)
	a.gauges["TotalAlloc"] = float64(ms.TotalAlloc)
	a.gauges["RandomValue"] = rand.Float64()

	a.pollCount++
}

func (a *Agent) Report() {
	a.mu.Lock()
	gauges := make(map[string]float64, len(a.gauges))
	for k, v := range a.gauges {
		gauges[k] = v
	}
	pollCount := a.pollCount
	a.mu.Unlock()

	for name, value := range gauges {
		url := fmt.Sprintf("%s/update/gauge/%s/%g", a.serverAddr, name, value)
		a.sendMetric(url)
	}

	url := fmt.Sprintf("%s/update/counter/PollCount/%d", a.serverAddr, pollCount)
	a.sendMetric(url)
}

func (a *Agent) sendMetric(url string) {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := a.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
