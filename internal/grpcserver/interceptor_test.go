package grpcserver

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTrustedSubnetInterceptor(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("parse cidr: %v", err)
	}

	interceptor := TrustedSubnetInterceptor(subnet)
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{}

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr codes.Code // codes.OK means no error
	}{
		{
			name:    "in subnet",
			ctx:     metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataRealIP, "192.168.1.10")),
			wantErr: codes.OK,
		},
		{
			name:    "out of subnet",
			ctx:     metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataRealIP, "10.0.0.1")),
			wantErr: codes.PermissionDenied,
		},
		{
			name:    "no metadata",
			ctx:     context.Background(),
			wantErr: codes.PermissionDenied,
		},
		{
			name:    "missing header",
			ctx:     metadata.NewIncomingContext(context.Background(), metadata.MD{}),
			wantErr: codes.PermissionDenied,
		},
		{
			name:    "invalid ip",
			ctx:     metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataRealIP, "bad")),
			wantErr: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interceptor(tt.ctx, nil, info, handler)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if status.Code(err) != tt.wantErr {
				t.Errorf("got code %v, want %v", status.Code(err), tt.wantErr)
			}
		})
	}
}
