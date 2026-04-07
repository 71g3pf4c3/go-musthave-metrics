package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	mu      sync.RWMutex
	ctx     context.Context
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (ms *MemStorage) AddCounter(_ context.Context, key string, value int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counter[key] += value
	return nil
}

func (ms *MemStorage) GetAllGauge(_ context.Context) (map[string]float64, error) {
	return ms.gauge, nil
}

func (ms *MemStorage) Ping(_ context.Context) error {
	return nil
}

func (ms *MemStorage) GetAllCounter(_ context.Context) (map[string]int64, error) {
	return ms.counter, nil
}

func (ms *MemStorage) SetGauge(_ context.Context, key string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauge[key] = value
	return nil
}

func (ms *MemStorage) GetValue(_ context.Context, name string, kind string) (string, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	switch kind {
	case models.Gauge:
		if value, ok := ms.gauge[name]; ok {
			return strconv.FormatFloat(value, 'f', -1, 64), nil
		}
		return "", ErrNotFound
	case models.Counter:
		if value, ok := ms.counter[name]; ok {
			return strconv.FormatInt(value, 10), nil
		}
		return "", ErrNotFound
	}
	return "", fmt.Errorf("UnexpectedError")
}

func (ms *MemStorage) GetGauge(_ context.Context, name string) (float64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if value, ok := ms.gauge[name]; ok {
		return value, nil
	}
	return 0, ErrNotFound
}

func (ms *MemStorage) GetCounter(_ context.Context, name string) (int64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if value, ok := ms.counter[name]; ok {
		return value, nil
	}
	return 0, ErrNotFound
}

func (ms *MemStorage) Snapshot() []models.Metrics {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	snap := make([]models.Metrics, 0, len(ms.gauge)+len(ms.counter))
	for id, value := range ms.gauge {
		snap = append(snap, models.Metrics{ID: id, MType: models.Gauge, Value: &value})
	}
	for id, value := range ms.counter {
		snap = append(snap, models.Metrics{ID: id, MType: models.Counter, Delta: &value})
	}
	return snap
}

func (ms *MemStorage) Dump(_ context.Context, path string) error {
	snap := ms.Snapshot()
	logger.Sugar.Infof("dumping %d metrics to %s", len(snap), path)
	data, err := json.MarshalIndent(snap, "", "   ")
	if err != nil {
		logger.Sugar.Debugf("failed to marshal metrics: %v", err)
		return err
	}
	return os.WriteFile(path, data, 0666)
}

func (ms *MemStorage) Restore(_ context.Context, path string) error {
	logger.Sugar.Infof("restoring metrics from %s", path)
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Sugar.Debugf("failed to read file %s: %v", path, err)
		return err
	}

	var metrics []models.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		logger.Sugar.Debugf("failed to unmarshal metrics: %v", err)
		return err
	}

	ms.gauge = make(map[string]float64)
	ms.counter = make(map[string]int64)

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				ms.gauge[metric.ID] = *metric.Value
			}
		case models.Counter:
			if metric.Delta != nil {
				ms.counter[metric.ID] = *metric.Delta
			}
		}
	}

	logger.Sugar.Infof("restored %d metrics", len(metrics))
	return nil
}

func (ms *MemStorage) UpdateBatch(_ context.Context, metrics []models.Metrics) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("invalid gauge metric")
			}
			ms.gauge[metric.ID] = *metric.Value
		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("invalid counter metric")
			}
			ms.counter[metric.ID] += *metric.Delta
		default:
			return fmt.Errorf("unsupported metric type")
		}
	}

	return nil
}
