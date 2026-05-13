package config

import (
	"log/slog"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg, err := LoadConfig("../../config.yaml.example", logger)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.HTTPListenPort == 0 {
		t.Error("expected HTTPListenPort to be set")
	}
	if cfg.LocalRouterID == "" {
		t.Error("expected LocalRouterID to be set")
	}
	if cfg.LocalASN == 0 {
		t.Error("expected LocalASN to be set")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	_, err := LoadConfig("nonexistent.yaml", logger)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestStripUnwanted(t *testing.T) {
	cfg := &Conf{
		NodeNameStripPatterns: []string{`re0\.`},
	}
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{"regex match", "re0.test", "test"},
		{"regex no match", "re0-test", "re0-test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if got := cfg.StripUnwanted(tt.arg); got != tt.want {
				t.Errorf("StripUnwanted() = %v, want %v", got, tt.want)
			}
		})
	}
}
