package utils

import (
	"os"
	"regexp"

	"github.com/neverthenetwork/inventa/src/inventa/logging"
	"gopkg.in/yaml.v2"
)

// Conf holds configuration information
type Conf struct {
	LocalJSONFile         string   `yaml:"local_json_file"`
	LocalRouterID         string   `yaml:"local_router_id"`
	LocalASN              int      `yaml:"local_asn"`
	PeerIPv4Address       string   `yaml:"peer_ipv4_address"`
	PeerASN               int      `yaml:"peer_asn"`
	HTTPListenPort        int      `yaml:"http_listen_port"`
	HTTPSEnable           bool     `yaml:"enable_https"`
	HTTPSCertFile         string   `yaml:"https_cert_file"`
	HTTPSKeyFile          string   `yaml:"https_key_file"`
	NodeNameStripPatterns []string `yaml:"node_name_strip_patterns" default:"[]"`
	GroupSplitChar        string   `yaml:"group_split_char" default:""`
	GroupSplitIndex       int      `yaml:"group_split_index" default:"0"`
}

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
func InitConfig(fileName string) {
	yamlFile, err := os.ReadFile(fileName)
	if err != nil {
		logging.Log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, &Configs)
	if err != nil {
		logging.Log.Fatal(err)
	}
}

// StripUnwanted removes any substrings from our name string
func StripUnwanted(name string) string {
	for _, pattern := range Configs.NodeNameStripPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			logging.Log.Fatal(err)
		}
		name = re.ReplaceAllString(name, "")
	}
	return name
}
