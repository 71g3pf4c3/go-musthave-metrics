package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
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

func New(config config.AgentConfig) *Agent {
	logger.Sugar.Infof("initializing agent, server=%s, poll=%ds, report=%ds",
		config.Address, config.PollInterval, config.ReportInterval)
	return &Agent{
		client:         resty.New(),
		serverAddr:     config.Address,
		gauges:         make(map[string]float64),
		pollInterval:   config.PollInterval,
		reportInterval: config.ReportInterval,
	}
}

func (a *Agent) Collect() {
	a.m.Lock()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	logger.Sugar.Debug("collecting runtime metrics")

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
	logger.Sugar.Debugf("metrics collected, pollCount=%d", a.pollCount)
	a.m.Unlock()
}

func (a *Agent) Report() {
	a.m.Lock()
	gauges := make(map[string]float64, len(a.gauges))
	maps.Copy(gauges, a.gauges)
	pollCount := a.pollCount
	a.m.Unlock()

	batch := make([]models.Metrics, 0, len(gauges)+1)
	for name, value := range gauges {
		v := value
		batch = append(batch, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}

	batch = append(batch, models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: &pollCount,
	})

	if len(batch) == 0 {
		logger.Sugar.Debug("skip empty batch")
		return
	}

	logger.Sugar.Infof("reporting %d metrics in batch", len(batch))
	if err := a.sendBatch(batch); err != nil {
		logger.Sugar.Errorf("send batch failed, fallback to single metrics: %v", err)
		for _, metric := range batch {
			if err := a.sendMetric(metric); err != nil {
				logger.Sugar.Errorf("fallback send %s/%s: %v", metric.MType, metric.ID, err)
			}
		}
	}

	logger.Sugar.Debug("report complete")
}

func (a *Agent) sendBatch(metrics []models.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("gzip write batch: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close batch: %w", err)
	}

	resp, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(buf.Bytes()).
		Post(fmt.Sprintf("%s/updates", a.serverAddr))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= http.StatusBadRequest {
		return fmt.Errorf("batch request failed with status %d", resp.StatusCode())
	}

	return nil
}

func (a *Agent) sendMetric(metric models.Metrics) error {
	logger.Sugar.Debugf("sending metric %s/%s", metric.MType, metric.ID)
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	resp, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(buf.Bytes()).
		Post(fmt.Sprintf("%s/update", a.serverAddr))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= http.StatusBadRequest {
		return fmt.Errorf("single metric request failed with status %d", resp.StatusCode())
	}

	return nil
}

func (a *Agent) Run() {
	logger.Sugar.Info("agent started")
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
