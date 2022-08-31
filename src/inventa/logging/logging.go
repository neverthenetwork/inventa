package logging

import (
	"github.com/osrg/gobgp/v3/pkg/log"
	"github.com/sirupsen/logrus"
)

// Log is the logging object
var Log *logrus.Logger

// SetUpLogger sets up the logger variable
func SetUpLogger() {
	Log = logrus.New()
	Log.SetLevel(logrus.InfoLevel)
	Log.Info("Set Logger Up")
}

// MyLogger implements github.com/osrg/gobgp/v3/pkg/log/Logger interface
type MyLogger struct {
	Logger *logrus.Logger
}

// Panic level
func (l *MyLogger) Panic(msg string, fields log.Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Panic(msg)
}

// Fatal Level
func (l *MyLogger) Fatal(msg string, fields log.Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Fatal(msg)
}

// Error level
func (l *MyLogger) Error(msg string, fields log.Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Error(msg)
}

// Warn level
func (l *MyLogger) Warn(msg string, fields log.Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Warn(msg)
}

// Info level
func (l *MyLogger) Info(msg string, fields log.Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Info(msg)
}

// Debug level
func (l *MyLogger) Debug(msg string, fields log.Fields) {
	l.Logger.WithFields(logrus.Fields(fields)).Debug(msg)
}

// SetLevel sets the level
func (l *MyLogger) SetLevel(level log.LogLevel) {
	l.Logger.SetLevel(logrus.Level(level))
}

// GetLevel gets the level
func (l *MyLogger) GetLevel() log.LogLevel {
	return log.LogLevel(l.Logger.GetLevel())
}
