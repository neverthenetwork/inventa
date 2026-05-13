package config

import (
	"testing"

	"github.com/neverthenetwork/inventa/internal/logging"
)

func TestInitConfig(t *testing.T) {
	logging.SetUpLogger()

	t.Run("valid config", func(_ *testing.T) {
		InitConfig("../../config.yaml.example")
		if Configs.HTTPListenPort == 0 {
			t.Error("expected HTTPListenPort to be set")
		}
		if Configs.LocalRouterID == "" {
			t.Error("expected LocalRouterID to be set")
		}
	})

	// Note: InitConfig calls log.Fatal on error (which exits the process),
	// so we can't test the error case here. This will be fixed in a future
	// refactor where InitConfig returns an error instead.
}

func TestStripUnwanted(t *testing.T) {
	Configs.NodeNameStripPatterns = []string{`re0\.`}
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
			if got := StripUnwanted(tt.arg); got != tt.want {
				t.Errorf("StripUnwanted() = %v, want %v", got, tt.want)
			}
		})
	}
}
