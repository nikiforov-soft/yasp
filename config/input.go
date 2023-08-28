package config

type Input struct {
	Transforms []*Transform  `yaml:"transforms"`
	Mqtt       *MqttInput    `yaml:"mqtt"`
	Memphis    *MemphisInput `yaml:"memphis"`
}
