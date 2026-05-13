package logging

import (
	"log/slog"
	"os"

	"github.com/osrg/gobgp/v3/pkg/log"
)

// NewLogger creates a new slog logger at Info level.
func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SlogAdapter adapts slog.Logger to the gobgp log.Logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a gobgp-compatible logger from a slog.Logger.
func NewSlogAdapter(l *slog.Logger) *SlogAdapter {
	return &SlogAdapter{logger: l}
}

func (a *SlogAdapter) toAttrs(fields log.Fields) []any {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return attrs
}

func (a *SlogAdapter) Panic(msg string, fields log.Fields) {
	a.logger.Error(msg, a.toAttrs(fields)...)
	panic(msg)
}

func (a *SlogAdapter) Fatal(msg string, fields log.Fields) {
	a.logger.Error(msg, a.toAttrs(fields)...)
	os.Exit(1)
}

func (a *SlogAdapter) Error(msg string, fields log.Fields) {
	a.logger.Error(msg, a.toAttrs(fields)...)
}

func (a *SlogAdapter) Warn(msg string, fields log.Fields) {
	a.logger.Warn(msg, a.toAttrs(fields)...)
}

func (a *SlogAdapter) Info(msg string, fields log.Fields) {
	a.logger.Info(msg, a.toAttrs(fields)...)
}

func (a *SlogAdapter) Debug(msg string, fields log.Fields) {
	a.logger.Debug(msg, a.toAttrs(fields)...)
}

func (a *SlogAdapter) SetLevel(_ log.LogLevel) {
	// slog levels are inverted vs gobgp — skip for now
}

func (a *SlogAdapter) GetLevel() log.LogLevel {
	return log.InfoLevel
}
