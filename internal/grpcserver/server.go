// Package grpcserver implements the gRPC Metrics service.
package grpcserver

import (
	"context"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	pb "github.com/71g3pf4c3/go-musthave-metrics/internal/proto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

// MetricsServer implements pb.MetricsServer backed by the metrics service.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	svc service.Service
}

// New creates a MetricsServer.
func New(svc service.Service) *MetricsServer {
	return &MetricsServer{svc: svc}
}

// UpdateMetrics accepts a batch of metrics and stores them.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	batch := make([]models.Metrics, 0, len(req.GetMetrics()))
	for _, m := range req.GetMetrics() {
		metric := models.Metrics{ID: m.GetId()}
		switch m.GetType() {
		case pb.Metric_COUNTER:
			metric.MType = models.Counter
			delta := m.GetDelta()
			metric.Delta = &delta
		default:
			metric.MType = models.Gauge
			value := m.GetValue()
			metric.Value = &value
		}
		batch = append(batch, metric)
	}

	if err := s.svc.BatchUpdate(ctx, batch); err != nil {
		return nil, err
	}
	return &pb.UpdateMetricsResponse{}, nil
}
