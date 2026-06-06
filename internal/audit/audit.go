// Package audit logs metric write events using the Observer pattern.
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
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
type Notifier struct {
	observers []Observer
}

// NewNotifier creates a Notifier with the given observers.
func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

// Notify sends the event to all observers.
func (n *Notifier) Notify(e Event) {
	for _, o := range n.observers {
		o.Notify(e)
	}
}

// FileObserver appends events as newline-delimited JSON to a file.
type FileObserver struct {
	path string
}

// NewFileObserver creates a FileObserver that writes to path.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify appends the event to the file.
func (f *FileObserver) Notify(e Event) {
	data, err := json.Marshal(e)
	if err != nil {
		logger.Sugar.Errorf("audit file: marshal event: %v", err)
		return
	}

	file, err := os.OpenFile(f.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logger.Sugar.Errorf("audit file: open %s: %v", f.path, err)
		return
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "%s\n", data); err != nil {
		logger.Sugar.Errorf("audit file: write: %v", err)
	}
}

// HTTPObserver sends events as JSON POST requests to a remote URL.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTPObserver that posts events to url.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
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
