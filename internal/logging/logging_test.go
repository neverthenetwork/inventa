package logging

import (
	"log/slog"
	"os"
	"testing"

	"github.com/osrg/gobgp/v3/pkg/log"
)

func TestNewLogger(t *testing.T) {
	l := NewLogger()
	if l == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestNewSlogAdapter(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	if a == nil {
		t.Fatal("NewSlogAdapter() returned nil")
	}
}

func TestSlogAdapter_toAttrs(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)

	fields := log.Fields{"key1": "val1", "key2": 42}
	attrs := a.toAttrs(fields)

	if len(attrs) != 4 {
		t.Fatalf("expected 4 attrs (2 key-value pairs), got %d", len(attrs))
	}
	// The order may vary since map iteration is non-deterministic,
	// so just check all expected keys and values are present
	found := make(map[string]bool)
	for i := 0; i < len(attrs); i += 2 {
		k := attrs[i].(string)
		found[k] = true
	}
	if !found["key1"] || !found["key2"] {
		t.Errorf("expected keys key1 and key2, got %v", found)
	}
}

func TestSlogAdapter_Info(_ *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	// Info should not panic
	a.Info("test message", log.Fields{"key": "value"})
}

func TestSlogAdapter_Debug(_ *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	a.Debug("debug message", log.Fields{})
}

func TestSlogAdapter_Warn(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	a.Warn("warning", log.Fields{"code": 404})
}

func TestSlogAdapter_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	a.Error("error occurred", log.Fields{"err": "something broke"})
}

func TestSlogAdapter_GetLevel(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	level := a.GetLevel()
	if level != log.InfoLevel {
		t.Errorf("expected InfoLevel, got %v", level)
	}
}

func TestSlogAdapter_SetLevel(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	// SetLevel is a no-op; should not panic
	a.SetLevel(log.DebugLevel)
}

func TestSlogAdapter_emptyFields(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := NewSlogAdapter(l)
	attrs := a.toAttrs(nil)
	if len(attrs) != 0 {
		t.Errorf("expected 0 attrs for nil fields, got %d", len(attrs))
	}
}
