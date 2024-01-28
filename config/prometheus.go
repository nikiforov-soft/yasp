package config

type Prometheus struct {
	Enabled        bool                       `yaml:"enabled"`
	ListenAddr     string                     `yaml:"listenAddr"`
	Endpoint       string                     `yaml:"endpoint"`
	TLS            PrometheusTlsConfig        `yaml:"tls"`
	MetricsMapping []PrometheusMetricsMapping `yaml:"metricsMapping"`
}

type PrometheusTlsConfig struct {
	CertificateFile string `yaml:"certificateFile"`
	PrivateKeyFile  string `yaml:"privateKeyFile"`
}

type PrometheusMetricsMapping struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Namespace   string            `yaml:"namespace"`
	Subsystem   string            `yaml:"subsystem"`
	Labels      map[string]string `yaml:"labels"`
	Type        string            `yaml:"type"`
	Value       string            `yaml:"value"`
	Condition   string            `yaml:"condition"`
}
