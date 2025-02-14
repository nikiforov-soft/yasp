package config

type Prometheus struct {
	Enabled        bool                       `yaml:"enabled"`
	MetricsMapping []PrometheusMetricsMapping `yaml:"metricsMapping"`
}

type PrometheusMetricsMapping struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Subsystem string            `yaml:"subsystem"`
	Labels    map[string]string `yaml:"labels"`
	Value     string            `yaml:"value"`
	Condition string            `yaml:"condition"`
}
