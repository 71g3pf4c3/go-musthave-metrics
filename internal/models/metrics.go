// Package models contains data types for metrics.
package models

// Metric type constants.
const (
	// Counter is a metric that only grows.
	Counter = "counter"
	// Gauge is a metric that can be set to any value.
	Gauge = "gauge"
)

// generate:reset
//
// Metrics is a single metric. Used in JSON requests and responses.
// Delta is set for counter, Value — for gauge.
type Metrics struct {
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // type: "counter" or "gauge"
	Delta *int64   `json:"delta,omitempty"` // counter value
	Value *float64 `json:"value,omitempty"` // gauge value
	Hash  string   `json:"hash,omitempty"`  // HMAC signature
}
