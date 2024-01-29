package config

type Metrics struct {
	Enabled    bool             `yaml:"enabled"`
	ListenAddr string           `yaml:"listenAddr"`
	Endpoint   string           `yaml:"endpoint"`
	TLS        MetricsTlsConfig `yaml:"tls"`
}

type MetricsTlsConfig struct {
	CertificateFile string `yaml:"certificateFile"`
	PrivateKeyFile  string `yaml:"privateKeyFile"`
}
