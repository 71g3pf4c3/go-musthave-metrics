// Package ipfilter provides middleware to restrict access to a trusted subnet.
package ipfilter

import (
	"net"
	"net/http"
)

// HeaderRealIP is the request header carrying the agent's real IP address.
const HeaderRealIP = "X-Real-IP"

// Middleware returns a middleware that only allows requests whose X-Real-IP
// header falls within the provided trusted subnet (CIDR). Requests from
// outside the subnet receive 403 Forbidden.
func Middleware(subnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := net.ParseIP(r.Header.Get(HeaderRealIP))
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
