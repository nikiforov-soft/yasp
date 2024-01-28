package config

type Output struct {
	Mqtt       *MqttOutput  `yaml:"mqtt"`
	InfluxDb2  *InfluxDb2   `yaml:"influxdb2"`
	Prometheus *Prometheus  `yaml:"prometheus"`
	Transforms []*Transform `yaml:"transforms"`
}
