package config

type Sensor struct {
	Name    string    `yaml:"name"`
	Input   *Input    `yaml:"input"`
	Outputs []*Output `yaml:"outputs"`
	Devices []*Device `yaml:"devices"`
}
