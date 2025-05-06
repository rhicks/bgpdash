package pkg

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	BGP struct {
		Local struct {
			RouterID string `yaml:"routerId"`
			ASN      int    `yaml:"asn"`
		} `yaml:"local"`
		Remote struct {
			PeerIP string `yaml:"peerIP"`
			ASN    int    `yaml:"asn"`
		} `yaml:"remote"`
	} `yaml:"bgp"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
