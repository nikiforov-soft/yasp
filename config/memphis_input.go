package config

import (
	"time"
)

type MemphisInput struct {
	Enabled        bool              `yaml:"enabled"`
	Hostname       string            `yaml:"hostname"`
	Username       string            `yaml:"username"`
	Password       string            `yaml:"password"`
	Station        string            `yaml:"station"`
	ConsumerName   string            `yaml:"consumerName"`
	PollInterval   time.Duration     `yaml:"pollInterval"`
	BatchSize      int               `yaml:"batchSize"`
	HeaderPrefixes map[string]string `yaml:"headerPrefixes"`
}
