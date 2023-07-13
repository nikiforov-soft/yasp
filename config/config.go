package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sensors []*Sensor `yaml:"sensors"`
}

func Load(filePath string) (*Config, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config *Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}
