// Package repository defines the storage interface and its implementations.
package repository

import (
	"context"
	"fmt"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

// Repository is the storage interface for metrics.
type Repository interface {
	// AddCounter adds value to counter key.
	AddCounter(ctx context.Context, key string, value int64) error
	// SetGauge sets gauge key to value.
	SetGauge(ctx context.Context, key string, value float64) error
	// GetValue returns metric value as string. ErrNotFound if missing.
	GetValue(ctx context.Context, key string, kind string) (string, error)
	// GetGauge returns float64 value of a gauge. ErrNotFound if missing.
	GetGauge(ctx context.Context, key string) (float64, error)
	// GetCounter returns int64 value of a counter. ErrNotFound if missing.
	GetCounter(ctx context.Context, key string) (int64, error)
	// GetAllGauge returns a copy of all gauges.
	GetAllGauge(ctx context.Context) (map[string]float64, error)
	// GetAllCounter returns a copy of all counters.
	GetAllCounter(ctx context.Context) (map[string]int64, error)
	// Dump saves all metrics to a file.
	Dump(ctx context.Context, path string) error
	// Ping checks the storage connection.
	Ping(ctx context.Context) error
	// Restore loads metrics from a file.
	Restore(ctx context.Context, path string) error
	// UpdateBatch writes multiple metrics at once.
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

// ErrNotFound is returned when a metric does not exist.
var ErrNotFound = fmt.Errorf("ErrNotFound")
