package utils

import (
	"os"
	"regexp"

	"github.com/osrg/gobgp/v3/pkg/log"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Conf holds configuration information
type Conf struct {
	RunTimeMode           string   `yaml:"run_time_mode"`
	LocalJSONFile         string   `yaml:"local_json_file"`
	LocalRouterID         string   `yaml:"local_router_id"`
	LocalASN              int      `yaml:"local_asn"`
	PeerIPv4Address       string   `yaml:"peer_ipv4_address"`
	PeerASN               int      `yaml:"peer_asn"`
	HTTPListenPort        int      `yaml:"http_listen_port"`
	NodeNameStripPatterns []string `yaml:"node_name_strip_patterns"`
	GroupSplitChar        string   `yaml:"group_split_char"`
	GroupSplitIndex       int      `yaml:"group_split_index"`
}

// Log is the logging object
var Log *logrus.Logger

// Configs is our shared config object
var Configs Conf

// FindInArray finds an element in an array
func FindInArray(what string, where []string) (idx int, found bool) {
	for i, v := range where {
		if v == what {
			return i, true
		}
	}
	return 0, false
}

// InitConfig initializes the configuration object
func InitConfig() {
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		Log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, &Configs)
	if err != nil {
		Log.Fatal(err)
	}
}

// SetUpLogger sets up the logger variable
func SetUpLogger() {
	Log = logrus.New()
	Log.SetLevel(logrus.InfoLevel)
	Log.Info("Set Logger Up")
}

// StripUnwanted removes any substrings from our name string
func StripUnwanted(name string) string {
	for _, pattern := range Configs.NodeNameStripPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			Log.Fatal(err)
		}
		name = re.ReplaceAllString(name, "")
	}
	return name
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
