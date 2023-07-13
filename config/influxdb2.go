package config

type InfluxDb2 struct {
	Enabled        bool              `yaml:"enabled"`
	Url            string            `yaml:"url"`
	AuthToken      string            `yaml:"authToken"`
	OrganizationId string            `yaml:"organizationId"`
	Bucket         string            `yaml:"bucket"`
	Measurement    string            `yaml:"measurement"`
	UseGZip        bool              `yaml:"useGZip"`
	BatchSize      uint              `yaml:"batchSize"`
	TagMapping     map[string]string `yaml:"tagMapping"`
	FieldMapping   map[string]string `yaml:"fieldMapping"`
}
