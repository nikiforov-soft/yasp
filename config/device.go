package config

type Device struct {
	Name       string            `yaml:"name"`
	Type       string            `yaml:"type"`
	Properties map[string]string `yaml:"properties"`
}
