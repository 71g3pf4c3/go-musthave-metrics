package repository

import (
	"context"
	"fmt"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

type Repository interface {
	AddCounter(ctx context.Context, key string, value int64) error
	SetGauge(ctx context.Context, key string, value float64) error
	GetValue(ctx context.Context, key string, kind string) (string, error)
	GetGauge(ctx context.Context, key string) (float64, error)
	GetCounter(ctx context.Context, key string) (int64, error)
	GetAllGauge(ctx context.Context) (map[string]float64, error)
	GetAllCounter(ctx context.Context) (map[string]int64, error)
	Dump(ctx context.Context, path string) error
	Ping(ctx context.Context) error
	Restore(ctx context.Context, path string) error
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

var ErrNotFound = fmt.Errorf("ErrNotFound")
