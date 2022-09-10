package utils

import (
	"testing"
)

func TestInitConfig(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
	}{
		{"config test - pass", args{"../../../config.yaml.example"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitConfig(tt.args.fileName)
		})
	}
}

func TestStripUnwanted(t *testing.T) {
	Configs.NodeNameStripPatterns = []string{"re0\\."}
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"strip unwanted - regex match", args{"re0.test"}, "test"},
		{"strip unwanted - regex nomatch", args{"re0-test"}, "re0-test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripUnwanted(tt.args.name); got != tt.want {
				t.Errorf("StripUnwanted() = %v, want %v", got, tt.want)
			}
		})
	}
}
