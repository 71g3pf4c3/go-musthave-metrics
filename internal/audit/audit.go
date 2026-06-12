// Package audit logs metric write events using the Observer pattern.
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/hashicorp/go-retryablehttp"
)

// Event is emitted after each successful metric write.
type Event struct {
	TS        int64    `json:"ts"`         // unix timestamp
	Metrics   []string `json:"metrics"`    // metric names
	IPAddress string   `json:"ip_address"` // client IP
}

// Observer receives audit events.
type Observer interface {
	// Notify is called after each successful write.
	Notify(e Event)
}

// Notifier broadcasts an Event to all registered observers.
// Each observer is notified in a separate goroutine so a slow observer
// does not block others. At most 100 goroutines run concurrently.
type Notifier struct {
	observers []Observer
	sem       chan struct{}
}

// NewNotifier creates a Notifier with the given observers.
func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{
		observers: observers,
		sem:       make(chan struct{}, 100),
	}
}

// Notify sends the event to every observer concurrently.
func (n *Notifier) Notify(e Event) {
	for _, o := range n.observers {
		o := o
		n.sem <- struct{}{}
		go func() {
			defer func() { <-n.sem }()
			o.Notify(e)
		}()
	}
}

// FileObserver appends events as newline-delimited JSON to a file.
// The file is kept open for the lifetime of the observer; call Close when done.
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver opens path for appending and returns a FileObserver.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("audit file: open %s: %w", path, err)
	}
	return &FileObserver{file: f}, nil
}

// Notify appends the event to the file.
func (f *FileObserver) Notify(e Event) {
	data, err := json.Marshal(e)
	if err != nil {
		logger.Sugar.Errorf("audit file: marshal event: %v", err)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := fmt.Fprintf(f.file, "%s\n", data); err != nil {
		logger.Sugar.Errorf("audit file: write: %v", err)
	}
}

// Close closes the underlying file.
func (f *FileObserver) Close() error {
	return f.file.Close()
}

// HTTPObserver sends events as JSON POST requests to a remote URL.
// Failed requests are retried automatically (up to 4 attempts with backoff).
type HTTPObserver struct {
	url    string
	client *retryablehttp.Client
}

// NewHTTPObserver creates an HTTPObserver that posts events to url.
func NewHTTPObserver(url string) *HTTPObserver {
	c := retryablehttp.NewClient()
	c.RetryMax = 3
	c.RetryWaitMin = time.Second
	c.RetryWaitMax = 5 * time.Second
	c.Logger = nil // silence default stderr logging
	return &HTTPObserver{url: url, client: c}
}

// Notify sends the event via HTTP POST.
func (h *HTTPObserver) Notify(e Event) {
	data, err := json.Marshal(e)
	if err != nil {
		logger.Sugar.Errorf("audit http: marshal event: %v", err)
		return
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Sugar.Errorf("audit http: post to %s: %v", h.url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		logger.Sugar.Errorf("audit http: server returned %d", resp.StatusCode)
	}
}
