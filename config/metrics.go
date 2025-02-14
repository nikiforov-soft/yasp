package config

import (
	"time"
)

type Metrics struct {
	Enabled           bool              `yaml:"enabled"`
	ListenAddr        string            `yaml:"listenAddr"`
	Endpoint          string            `yaml:"endpoint"`
	TLS               MetricsTlsConfig  `yaml:"tls"`
	MetricsMapping    []*MetricsMapping `yaml:"metricsMapping"`
	StalenessInterval time.Duration     `yaml:"stalenessInterval"`
}

type MetricsTlsConfig struct {
	CertificateFile string `yaml:"certificateFile"`
	PrivateKeyFile  string `yaml:"privateKeyFile"`
}
