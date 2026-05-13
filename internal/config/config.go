package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Conf holds configuration information.
type Conf struct {
	LocalJSONFile         string   `yaml:"local_json_file"`
	LocalRouterID         string   `yaml:"local_router_id"`
	LocalASN              int      `yaml:"local_asn"`
	PeerIPv4Address       string   `yaml:"peer_ipv4_address"`
	PeerASN               int      `yaml:"peer_asn"`
	HTTPListenPort        int      `yaml:"http_listen_port"`
	HTTPSEnable           bool     `yaml:"https_enable"`
	HTTPSCertFile         string   `yaml:"https_cert_file"`
	HTTPSKeyFile          string   `yaml:"https_key_file"`
	NodeNameStripPatterns []string `yaml:"node_name_strip_patterns" default:"[]"`
	GroupSplitChar        string   `yaml:"group_split_char" default:""`
	GroupSplitIndex       int      `yaml:"group_split_index" default:"0"`
}

// FindInArray returns the index of what in where, and whether it was found.
func FindInArray(what string, where []string) (idx int, found bool) {
	for i, v := range where {
		if v == what {
			return i, true
		}
	}
	return 0, false
}

// LoadConfig loads and returns the configuration from a YAML file.
func LoadConfig(fileName string, logger *slog.Logger) (*Conf, error) {
	yamlFile, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", fileName, err)
	}
	var cfg Conf
	if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	logger.Info("config loaded", "file", fileName)
	return &cfg, nil
}

// StripUnwanted removes configured patterns from a node name.
func (c *Conf) StripUnwanted(name string) string {
	for _, pattern := range c.NodeNameStripPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue // skip invalid patterns
		}
		name = re.ReplaceAllString(name, "")
	}
	return name
}
