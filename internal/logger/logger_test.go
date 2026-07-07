package logger_test

import (
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
)

func TestInitialize_ValidLevel(t *testing.T) {
	if err := logger.Initialize("info"); err != nil {
		t.Errorf("expected no error for valid level \"info\", got %v", err)
	}
}

func TestInitialize_InvalidLevel(t *testing.T) {
	if err := logger.Initialize("not-a-level"); err == nil {
		t.Error("expected error for invalid log level, got nil")
	}
}
