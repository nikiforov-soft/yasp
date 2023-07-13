package config

type Transform struct {
	Name       string            `yaml:"name"`
	Properties map[string]string `yaml:"properties"`
}
