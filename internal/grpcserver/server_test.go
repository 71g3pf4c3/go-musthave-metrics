package grpcserver

import (
	"context"
	"testing"

	pb "github.com/71g3pf4c3/go-musthave-metrics/internal/proto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

func TestUpdateMetrics(t *testing.T) {
	svc := service.New(repository.NewMemStorage())
	srv := New(svc)

	delta := int64(5)
	value := 3.14
	req := &pb.UpdateMetricsRequest{Metrics: []*pb.Metric{
		{Id: "hits", Type: pb.Metric_COUNTER, Delta: delta},
		{Id: "cpu", Type: pb.Metric_GAUGE, Value: value},
	}}

	if _, err := srv.UpdateMetrics(context.Background(), req); err != nil {
		t.Fatalf("UpdateMetrics: %v", err)
	}

	got, err := svc.GetValue(context.Background(), "counter", "hits")
	if err != nil {
		t.Fatalf("get counter: %v", err)
	}
	if got != "5" {
		t.Errorf("counter = %q, want 5", got)
	}

	got, err = svc.GetValue(context.Background(), "gauge", "cpu")
	if err != nil {
		t.Fatalf("get gauge: %v", err)
	}
	if got != "3.14" {
		t.Errorf("gauge = %q, want 3.14", got)
	}
}
