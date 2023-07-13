package config

type Input struct {
	Transforms []*Transform `yaml:"transforms"`
	Mqtt       *Mqtt        `yaml:"mqtt"`
}
