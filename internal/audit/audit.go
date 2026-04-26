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

// Event — формат события аудита.
type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer — интерфейс приёмника аудита (паттерн Наблюдатель).
type Observer interface {
	Notify(e Event)
}

// Notifier рассылает событие всем зарегистрированным наблюдателям.
type Notifier struct {
	observers []Observer
}

func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

func (n *Notifier) Notify(e Event) {
	for _, o := range n.observers {
		o.Notify(e)
	}
}

// FileObserver пишет события в файл (append, одна строка — одно событие).
type FileObserver struct {
	path string
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

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

// HTTPObserver отправляет события на удалённый сервер методом POST.
type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

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
