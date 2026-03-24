package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

type Repository interface {
	AddCounter(key string, value int64)
	SetGauge(key string, value float64)
	GetValue(key string) (string, error)
	GetGauge(key string) (float64, error)
	GetCounter(key string) (int64, error)
	GetAllGauge() (map[string]float64, error)
	GetAllCounter() (map[string]int64, error)
	Dump(path string) error
	Restore(path string) error
}

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	m       sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (ms *MemStorage) AddCounter(key string, value int64) {
	ms.m.Lock()
	ms.counter[key] += value
	ms.m.Unlock()
}

func (ms *MemStorage) GetAllGauge() map[string]float64 {
	return ms.gauge
}

func (ms *MemStorage) GetAllCounter() map[string]int64 {
	return ms.counter
}

func (ms *MemStorage) SetGauge(key string, value float64) {
	ms.m.Lock()
	ms.gauge[key] = value
	ms.m.Unlock()
}

var ErrNotFound = fmt.Errorf("ErrNotFound")

func (ms *MemStorage) GetValue(name string, kind string) (string, error) {
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

func (ms *MemStorage) GetGauge(name string) (float64, error) {
	if value, ok := ms.gauge[name]; ok {
		return value, nil
	}
	return 0, ErrNotFound
}

func (ms *MemStorage) GetCounter(name string) (int64, error) {
	if value, ok := ms.counter[name]; ok {
		return value, nil
	}
	return 0, ErrNotFound
}

func (ms *MemStorage) Snapshot() []models.Metrics {
	ms.m.Lock()
	defer ms.m.Unlock()
	snap := make([]models.Metrics, 0, len(ms.gauge)+len(ms.counter))
	for id, value := range ms.gauge {
		snap = append(snap, models.Metrics{ID: id, MType: models.Gauge, Value: &value})
	}
	for id, value := range ms.counter {
		snap = append(snap, models.Metrics{ID: id, MType: models.Counter, Delta: &value})
	}
	return snap
}

func (ms *MemStorage) Dump(path string) error {
	snap := ms.Snapshot()
	data, err := json.MarshalIndent(snap, "", "   ")
	if err != nil {
		return err
	}
	// сохраняем данные в файл
	return os.WriteFile(path, data, 0666)
}

func (ms *MemStorage) Restore(path string) error {
	ms.m.Lock()
	defer ms.m.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var metrics []models.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return err
	}

	// Clear existing data
	ms.gauge = make(map[string]float64)
	ms.counter = make(map[string]int64)

	// Restore metrics from file
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

	return nil
}
