package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MetadataRealIP is the metadata key carrying the agent's real IP address.
const MetadataRealIP = "x-real-ip"

// TrustedSubnetInterceptor returns a unary interceptor that rejects requests
// whose x-real-ip metadata is missing, malformed, or outside the trusted
// subnet with codes.PermissionDenied.
func TrustedSubnetInterceptor(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}

		values := md.Get(MetadataRealIP)
		if len(values) == 0 {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip")
		}

		ip := net.ParseIP(values[0])
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "ip is not in trusted subnet")
		}

		return handler(ctx, req)
	}
}
