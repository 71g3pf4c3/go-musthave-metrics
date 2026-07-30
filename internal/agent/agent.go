package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdh"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	agentcrypto "github.com/71g3pf4c3/go-musthave-metrics/internal/crypto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/ipfilter"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/sign"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"resty.dev/v3"
)

type Agent struct {
	client         *resty.Client
	serverAddr     string
	gauges         map[string]float64
	pollCount      int64
	pollInterval   int
	reportInterval int
	rateLimit      int
	key            string
	publicKey      *ecdh.PublicKey
	realIP         string
	m              sync.Mutex
}

var agentRetryDelays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

// outboundIP determines the agent host's outbound IP address used to reach
// external hosts. Falls back to loopback if it cannot be determined.
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func New(cfg config.AgentConfig) (*Agent, error) {
	rl := cfg.RateLimit
	if rl <= 0 {
		rl = 1
	}

	a := &Agent{
		client:         resty.New(),
		serverAddr:     cfg.Address,
		gauges:         make(map[string]float64),
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		rateLimit:      rl,
		key:            cfg.Key,
		realIP:         outboundIP(),
	}

	if cfg.CryptoKey != "" {
		pub, err := agentcrypto.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}
		a.publicKey = pub
		logger.Sugar.Infof("X25519 ECDH encryption enabled")
	}

	logger.Sugar.Infof("initializing agent, server=%s, poll=%ds, report=%ds", cfg.Address, cfg.PollInterval, cfg.ReportInterval)
	return a, nil
}

func (a *Agent) Collect() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	a.m.Lock()
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

func (a *Agent) CollectExtra() {
	v, verr := mem.VirtualMemory()
	c, cerr := cpu.Percent(0, true)

	a.m.Lock()
	if verr == nil {
		a.gauges["TotalMemory"] = float64(v.Total)
		a.gauges["FreeMemory"] = float64(v.Free)
	}
	if cerr == nil {
		for i, load := range c {
			a.gauges[fmt.Sprintf("CPUutilization%d", i+1)] = load
		}
	}
	a.m.Unlock()
}

func (a *Agent) BuildBatch() []models.Metrics {
	a.m.Lock()
	gauges := maps.Clone(a.gauges)
	pollCount := a.pollCount
	a.m.Unlock()

	batch := make([]models.Metrics, 0, len(gauges)+1)
	for name, value := range gauges {
		v := value
		batch = append(batch, models.Metrics{ID: name, MType: models.Gauge, Value: &v})
	}
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount})
	return batch
}

func (a *Agent) SendBatch(batch []models.Metrics) {
	ctx := context.Background()
	if err := a.sendBatch(ctx, batch); err != nil {
		logger.Sugar.Errorf("send batch failed, fallback to single: %v", err)
		for _, m := range batch {
			if err := a.sendMetric(ctx, m); err != nil {
				logger.Sugar.Errorf("fallback send %s/%s: %v", m.MType, m.ID, err)
			}
		}
	}
}

func (a *Agent) worker(jobs <-chan []models.Metrics) {
	for batch := range jobs {
		a.SendBatch(batch)
	}
}

func (a *Agent) Run(ctx context.Context) {
	logger.Sugar.Infof("agent started, workers=%d", a.rateLimit)

	var workerWg sync.WaitGroup
	var collectWg sync.WaitGroup

	jobs := make(chan []models.Metrics, a.rateLimit)
	for i := 0; i < a.rateLimit; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			a.worker(jobs)
		}()
	}

	pollTicker := time.NewTicker(time.Duration(a.pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.reportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Sugar.Infof("agent shutting down, waiting for collectors...")
			collectWg.Wait()

			logger.Sugar.Infof("sending final batch...")
			batch := a.BuildBatch()
			if len(batch) > 0 {
				a.SendBatch(batch)
			}
			close(jobs)
			workerWg.Wait()
			logger.Sugar.Infof("agent stopped")
			return
		case <-pollTicker.C:
			collectWg.Add(2)
			go func() {
				defer collectWg.Done()
				a.Collect()
			}()
			go func() {
				defer collectWg.Done()
				a.CollectExtra()
			}()
		case <-reportTicker.C:
			batch := a.BuildBatch()
			if len(batch) > 0 {
				jobs <- batch
			}
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

	body := buf.Bytes()
	if a.publicKey != nil {
		enc, encErr := agentcrypto.Encrypt(a.publicKey, body)
		if encErr != nil {
			return fmt.Errorf("encrypt batch: %w", encErr)
		}
		body = enc
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
			SetHeader(ipfilter.HeaderRealIP, a.realIP).
			SetBody(body)
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

	body := buf.Bytes()
	if a.publicKey != nil {
		enc, encErr := agentcrypto.Encrypt(a.publicKey, body)
		if encErr != nil {
			return fmt.Errorf("encrypt metric: %w", encErr)
		}
		body = enc
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
			SetHeader(ipfilter.HeaderRealIP, a.realIP).
			SetBody(body)
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
