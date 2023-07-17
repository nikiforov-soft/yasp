package config

type Sensor struct {
	Enabled bool      `yaml:"enabled"`
	Name    string    `yaml:"name"`
	Input   *Input    `yaml:"input"`
	Outputs []*Output `yaml:"outputs"`
	Devices []*Device `yaml:"devices"`
}
