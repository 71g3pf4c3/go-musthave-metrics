package repository

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

type Repository interface {
	AddCounter(key string, value int64)
	SetGauge(key string, value float64)
	GetValue(key string) (string, error)
	GetAllGauge() (map[string]float64, error)
	GetAllCounter() (map[string]int64, error)
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
