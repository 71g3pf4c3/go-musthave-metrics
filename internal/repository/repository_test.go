package repository

import (
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

func TestNewStorage(t *testing.T) {
	ms := NewMemStorage()
	if ms == nil {
		t.Fatal("expected non-nil MemStorage")
	}
}

func TestSetGauge(t *testing.T) {
	ms := NewMemStorage()
	ms.SetGauge("cpu", 42.5)

	v, err := ms.GetValue("cpu", models.Gauge)
	if err != nil {
		t.Fatalf("unexpected error reading gauge: %v", err)
	}
	if v != "42.5" {
		t.Errorf("expected \"42.5\", got %q", v)
	}
}

func TestSetGaugeWrong(t *testing.T) {
	ms := NewMemStorage()
	// Reading a key that was never set must return an error.
	_, err := ms.GetValue("nonexistent_gauge", models.Gauge)
	if err == nil {
		t.Error("expected error for nonexistent key, got nil")
	}
}

func TestAddNewCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.AddCounter("requests", 1)

	v, err := ms.GetValue("requests", models.Counter)
	if err != nil {
		t.Fatalf("unexpected error reading counter: %v", err)
	}
	if v != "1" {
		t.Errorf("expected \"1\", got %q", v)
	}
}

func TestAddNewCounterWrong(t *testing.T) {
	ms := NewMemStorage()
	// Reading a counter that was never set must return an error.
	_, err := ms.GetValue("nonexistent_counter", models.Counter)
	if err == nil {
		t.Error("expected error for nonexistent counter key, got nil")
	}
}

func TestAddExistingCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.AddCounter("hits", 10)
	ms.AddCounter("hits", 5)

	v, err := ms.GetValue("hits", models.Counter)
	if err != nil {
		t.Fatalf("unexpected error reading counter: %v", err)
	}
	if v != "15" {
		t.Errorf("expected \"15\" (10+5), got %q", v)
	}
}

func TestAddExistingCounterWrong(t *testing.T) {
	ms := NewMemStorage()
	// Gauge and counter use separate maps, so setting a gauge under "mixed"
	// and then adding a counter under the same key stores 5 in the counter map.
	ms.SetGauge("mixed", 99.9)
	ms.AddCounter("mixed", 5)

	v, err := ms.GetValue("mixed", models.Counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "5" {
		t.Errorf("expected \"5\", got %q", v)
	}
}

func TestGetValueGauge(t *testing.T) {
	ms := NewMemStorage()
	ms.SetGauge("cpu", 42.5)

	v, err := ms.GetValue("cpu", models.Gauge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "42.5" {
		t.Errorf("expected \"42.5\", got %q", v)
	}
}

func TestGetValueCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.AddCounter("requests", 100)

	v, err := ms.GetValue("requests", models.Counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "100" {
		t.Errorf("expected \"100\", got %q", v)
	}
}

func TestGetValueCounterAccumulated(t *testing.T) {
	ms := NewMemStorage()
	ms.AddCounter("hits", 10)
	ms.AddCounter("hits", 5)

	v, err := ms.GetValue("hits", models.Counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "15" {
		t.Errorf("expected \"15\", got %q", v)
	}
}

func TestGetValueGaugeNotFound(t *testing.T) {
	ms := NewMemStorage()

	_, err := ms.GetValue("nonexistent", models.Gauge)
	if err == nil {
		t.Error("expected error for nonexistent gauge key, got nil")
	}
}

func TestGetValueCounterNotFound(t *testing.T) {
	ms := NewMemStorage()

	_, err := ms.GetValue("nonexistent", models.Counter)
	if err == nil {
		t.Error("expected error for nonexistent counter key, got nil")
	}
}

func TestGetValueUnknownKind(t *testing.T) {
	ms := NewMemStorage()
	ms.SetGauge("cpu", 1.0)

	_, err := ms.GetValue("cpu", "histogram")
	if err == nil {
		t.Error("expected error for unknown metric kind, got nil")
	}
}
