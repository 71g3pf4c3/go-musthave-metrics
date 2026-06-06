package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

func BenchmarkSetGauge(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.SetGauge(ctx, "cpu", float64(i))
	}
}

func BenchmarkAddCounter(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.AddCounter(ctx, "hits", 1)
	}
}

func BenchmarkGetValue(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()
	_ = ms.SetGauge(ctx, "cpu", 42.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ms.GetValue(ctx, "cpu", models.Gauge)
	}
}

func BenchmarkUpdateBatch(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	batch := make([]models.Metrics, 30)
	for i := range batch {
		v := float64(i)
		batch[i] = models.Metrics{ID: fmt.Sprintf("metric%d", i), MType: models.Gauge, Value: &v}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.UpdateBatch(ctx, batch)
	}
}

func BenchmarkSnapshot(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		v := float64(i)
		_ = ms.SetGauge(ctx, fmt.Sprintf("gauge%d", i), v)
		_ = ms.AddCounter(ctx, fmt.Sprintf("counter%d", i), int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.Snapshot()
	}
}
