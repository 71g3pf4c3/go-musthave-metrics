package repository

import (
	"testing"
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

	v, err := ms.Read("cpu")
	if err != nil {
		t.Fatalf("unexpected error reading gauge: %v", err)
	}
	got, ok := v.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", v)
	}
	if got != 42.5 {
		t.Errorf("expected 42.5, got %v", got)
	}
}

func TestSetGaugeWrong(t *testing.T) {
	ms := NewMemStorage()
	// Reading a key that was never set must return an error.
	_, err := ms.Read("nonexistent_gauge")
	if err == nil {
		t.Error("expected error for nonexistent key, got nil")
	}
}

func TestAddNewCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.AddCounter("requests", 1)

	v, err := ms.Read("requests")
	if err != nil {
		t.Fatalf("unexpected error reading counter: %v", err)
	}
	got, ok := v.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", v)
	}
	if got != 1 {
		t.Errorf("expected 1, got %v", got)
	}
}

func TestAddNewCounterWrong(t *testing.T) {
	ms := NewMemStorage()
	// Reading a counter that was never set must return an error.
	_, err := ms.Read("nonexistent_counter")
	if err == nil {
		t.Error("expected error for nonexistent counter key, got nil")
	}
}

func TestAddExistingCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.AddCounter("hits", 10)
	ms.AddCounter("hits", 5)

	v, err := ms.Read("hits")
	if err != nil {
		t.Fatalf("unexpected error reading counter: %v", err)
	}
	got, ok := v.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", v)
	}
	if got != 15 {
		t.Errorf("expected 15 (10+5), got %v", got)
	}
}

func TestAddExistingCounterWrong(t *testing.T) {
	ms := NewMemStorage()
	// When a gauge is stored under the same key, the type assertion inside
	// AddCounter fails and the value is replaced (not accumulated).
	ms.SetGauge("mixed", 99.9)
	ms.AddCounter("mixed", 5)

	v, err := ms.Read("mixed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := v.(int64)
	if !ok {
		t.Fatalf("expected int64 after counter replaced gauge, got %T", v)
	}
	if got != 5 {
		t.Errorf("expected 5 (no accumulation because previous type was float64), got %v", got)
	}
}
