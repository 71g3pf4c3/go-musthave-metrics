package repository

import "fmt"

type Repository interface {
	Write(key string, value interface{})
	AddCounter(key string, value int64)
	SetGauge(key string, value float64)
	Read(key string) (interface{}, error)
}

type MemStorage struct {
	data map[string]interface{}
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]interface{}),
	}
}

func (ms *MemStorage) Write(key string, value interface{}) {
	ms.data[key] = value
}

func (ms *MemStorage) AddCounter(key string, value int64) {
	if existing, ok := ms.data[key].(int64); ok {
		ms.data[key] = existing + value
	} else {
		ms.data[key] = value
	}
}

func (ms *MemStorage) SetGauge(key string, value float64) {
	ms.data[key] = value
}

func (ms *MemStorage) Read(key string) (interface{}, error) {
	v, ok := ms.data[key]
	if !ok {
		return 0, fmt.Errorf("ErrNotFound")
	}
	return v, nil
}
