package agent

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/grpcserver"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	pb "github.com/71g3pf4c3/go-musthave-metrics/internal/proto"
)

// grpcClient sends metric batches to the server over gRPC.
type grpcClient struct {
	conn   *grpc.ClientConn
	client pb.MetricsClient
	realIP string
}

// newGRPCClient dials the gRPC server and returns a ready-to-use client.
func newGRPCClient(target, realIP string) (*grpcClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &grpcClient{
		conn:   conn,
		client: pb.NewMetricsClient(conn),
		realIP: realIP,
	}, nil
}

// SendBatch converts and sends metrics as a single UpdateMetrics request,
// passing the agent's IP in the x-real-ip metadata.
func (c *grpcClient) SendBatch(ctx context.Context, batch []models.Metrics) error {
	metrics := make([]*pb.Metric, 0, len(batch))
	for _, m := range batch {
		pm := &pb.Metric{Id: m.ID}
		switch m.MType {
		case models.Counter:
			pm.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				pm.Delta = *m.Delta
			}
		default:
			pm.Type = pb.Metric_GAUGE
			if m.Value != nil {
				pm.Value = *m.Value
			}
		}
		metrics = append(metrics, pm)
	}

	ctx = metadata.AppendToOutgoingContext(ctx, grpcserver.MetadataRealIP, c.realIP)
	_, err := c.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{Metrics: metrics})
	return err
}

// Close releases the underlying connection.
func (c *grpcClient) Close() {
	_ = c.conn.Close()
}
