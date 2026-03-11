package agent

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"resty.dev/v3"
)

type Agent struct {
	client         *resty.Client
	serverAddr     string
	gauges         map[string]float64
	pollCount      int64
	pollInterval   int
	reportInterval int
	m              sync.Mutex
}

type AgentConfig struct {
	PollInterval   int
	ReportInterval int
}

func New(serverAddr string, config AgentConfig) *Agent {
	if !strings.HasPrefix(serverAddr, "http://") && !strings.HasPrefix(serverAddr, "https://") {
		serverAddr = "http://" + serverAddr
	}
	return &Agent{
		client:         resty.New(),
		serverAddr:     serverAddr,
		gauges:         make(map[string]float64),
		pollInterval:   config.PollInterval,
		reportInterval: config.ReportInterval,
	}
}

func (a *Agent) Collect() {
	a.m.Lock()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

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
	a.m.Unlock()
}

func (a *Agent) Report() {
	a.m.Lock()
	gauges := make(map[string]float64, len(a.gauges))
	for k, v := range a.gauges {
		gauges[k] = v
	}
	pollCount := a.pollCount
	a.m.Unlock()

	for name, value := range gauges {
		url := fmt.Sprintf("%s/update/gauge/%s/%s", a.serverAddr, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err := a.sendMetric(url); err != nil {
			fmt.Printf("send gauge %s: %v\n", name, err)
		}
	}

	url := fmt.Sprintf("%s/update/counter/PollCount/%d", a.serverAddr, pollCount)
	if err := a.sendMetric(url); err != nil {
		fmt.Printf("send counter PollCount: %v\n", err)
	}
}

func (a *Agent) sendMetric(url string) error {
	_, err := a.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)
	return err
}

func (a *Agent) Run() {
	pollTicker := time.NewTicker(time.Duration(a.pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.reportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()
	for {
		select {
		case <-pollTicker.C:
			a.Collect()
		case <-reportTicker.C:
			a.Report()
		}
	}
}
