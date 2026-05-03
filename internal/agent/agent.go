package agent

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"github.com/71g3pf4c3/go-musthave-metrics/internal/sign"
	"resty.dev/v3"
)

type Agent struct {
	client         *resty.Client
	serverAddr     string
	gauges         map[string]float64
	pollCount      int64
	pollInterval   int
	reportInterval int
	key            string
	m              sync.Mutex
}

var agentRetryDelays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

func New(config config.AgentConfig) *Agent {
	logger.Sugar.Infof("initializing agent, server=%s, poll=%ds, report=%ds",
		config.Address, config.PollInterval, config.ReportInterval)
	return &Agent{
		client:         resty.New(),
		serverAddr:     config.Address,
		gauges:         make(map[string]float64),
		pollInterval:   config.PollInterval,
		reportInterval: config.ReportInterval,
		key:            config.Key,
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
	a.report(context.Background())
}

func (a *Agent) report(ctx context.Context) {
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
	if err := a.sendBatch(ctx, batch); err != nil {
		logger.Sugar.Errorf("send batch failed, fallback to single metrics: %v", err)
		for _, metric := range batch {
			if err := a.sendMetric(ctx, metric); err != nil {
				logger.Sugar.Errorf("fallback send %s/%s: %v", metric.MType, metric.ID, err)
			}
		}
	}

	logger.Sugar.Debug("report complete")
}

func (a *Agent) sendBatch(ctx context.Context, metrics []models.Metrics) error {
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

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		req := a.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(buf.Bytes())
		if a.key != "" {
			req.SetHeader(sign.HeaderHashSHA256, sign.ComputeHMAC(data, a.key))
		}

		resp, err := req.Post(fmt.Sprintf("%s/updates", a.serverAddr))
		if err == nil {
			if resp.StatusCode() >= http.StatusBadRequest {
				return fmt.Errorf("batch request failed with status %d", resp.StatusCode())
			}
			return nil
		}

		if attempt >= len(agentRetryDelays) {
			return err
		}

		delay := agentRetryDelays[attempt]
		logger.Sugar.Infof("batch send retry attempt=%d in %s: %v", attempt+1, delay, err)
		if err := sleepCtx(ctx, delay); err != nil {
			return err
		}
	}
}

func (a *Agent) sendMetric(ctx context.Context, metric models.Metrics) error {
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

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		req := a.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(buf.Bytes())
		if a.key != "" {
			req.SetHeader(sign.HeaderHashSHA256, sign.ComputeHMAC(data, a.key))
		}

		resp, err := req.Post(fmt.Sprintf("%s/update", a.serverAddr))
		if err == nil {
			if resp.StatusCode() >= http.StatusBadRequest {
				return fmt.Errorf("single metric request failed with status %d", resp.StatusCode())
			}
			return nil
		}

		if attempt >= len(agentRetryDelays) {
			return err
		}

		delay := agentRetryDelays[attempt]
		logger.Sugar.Infof("single metric retry attempt=%d in %s: %v", attempt+1, delay, err)
		if err := sleepCtx(ctx, delay); err != nil {
			return err
		}
	}
}

func sleepCtx(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (a *Agent) Run(ctx context.Context) {
	logger.Sugar.Info("agent started")
	pollTicker := time.NewTicker(time.Duration(a.pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.reportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Sugar.Infof("agent stopped: %v", ctx.Err())
			return
		case <-pollTicker.C:
			a.Collect()
		case <-reportTicker.C:
			a.report(ctx)
		}
	}
}
