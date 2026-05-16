package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// SourceConfig holds configuration for an individual topology source.
type SourceConfig struct {
	Enabled bool `yaml:"enabled"`
}

// BGPSourceConfig holds BGP-LS specific configuration.
type BGPSourceConfig struct {
	SourceConfig    `yaml:",inline"`
	LocalRouterID   string `yaml:"local_router_id"`
	LocalASN        int    `yaml:"local_asn"`
	PeerIPv4Address string `yaml:"peer_ipv4_address"`
	PeerASN         int    `yaml:"peer_asn"`
}

// LocalJSONSourceConfig holds local JSON file source configuration.
type LocalJSONSourceConfig struct {
	SourceConfig `yaml:",inline"`
	File         string `yaml:"file"`
}

// AWSSourceConfig holds AWS topology discovery configuration.
type AWSSourceConfig struct {
	SourceConfig `yaml:",inline"`
	Regions      []string `yaml:"regions"`
	Profile      string   `yaml:"profile"`      // AWS profile name (optional)
	RoleARN      string   `yaml:"role_arn"`     // IAM role to assume (optional)
	EndpointURL  string   `yaml:"endpoint_url"` // Custom endpoint (e.g. Floci/LocalStack)
	PollInterval int      `yaml:"poll_interval_seconds" default:"300"`
}

// Sources holds per-source configuration.
type Sources struct {
	BGPLS     BGPSourceConfig       `yaml:"bgpls"`
	LocalJSON LocalJSONSourceConfig `yaml:"localjson"`
	AWS       AWSSourceConfig       `yaml:"aws"`
}

// Conf holds configuration information.
type Conf struct {
	// Legacy flat BGP fields — kept for backward compatibility with existing configs.
	// New configs should use sources.bgpls.* instead.
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

	// Sources holds per-source configuration. When a source section is present,
	// its fields take precedence over legacy flat fields.
	Sources Sources `yaml:"sources"`
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

	// Populate legacy fields from source config when not set at top level.
	// This allows new-style source configs to work with code that reads flat fields.
	if cfg.Sources.BGPLS.LocalRouterID != "" && cfg.LocalRouterID == "" {
		cfg.LocalRouterID = cfg.Sources.BGPLS.LocalRouterID
	}
	if cfg.Sources.BGPLS.LocalASN != 0 && cfg.LocalASN == 0 {
		cfg.LocalASN = cfg.Sources.BGPLS.LocalASN
	}
	if cfg.Sources.BGPLS.PeerIPv4Address != "" && cfg.PeerIPv4Address == "" {
		cfg.PeerIPv4Address = cfg.Sources.BGPLS.PeerIPv4Address
	}
	if cfg.Sources.BGPLS.PeerASN != 0 && cfg.PeerASN == 0 {
		cfg.PeerASN = cfg.Sources.BGPLS.PeerASN
	}
	if cfg.Sources.LocalJSON.File != "" && cfg.LocalJSONFile == "" {
		cfg.LocalJSONFile = cfg.Sources.LocalJSON.File
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
