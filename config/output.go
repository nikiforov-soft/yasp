package config

type Output struct {
	Mqtt       *Mqtt        `yaml:"mqtt"`
	Transforms []*Transform `yaml:"transforms"`
}
