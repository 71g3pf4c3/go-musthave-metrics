package ipfilter

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("parse cidr: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(subnet)(next)

	tests := []struct {
		name       string
		realIP     string
		wantStatus int
	}{
		{"in subnet", "192.168.1.42", http.StatusOK},
		{"out of subnet", "10.0.0.1", http.StatusForbidden},
		{"missing header", "", http.StatusForbidden},
		{"invalid ip", "not-an-ip", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/updates", nil)
			if tt.realIP != "" {
				req.Header.Set(HeaderRealIP, tt.realIP)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
