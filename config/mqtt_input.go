package config

import (
	"net/url"
)

type MqttInput struct {
	Enabled    bool     `yaml:"enabled"`
	Topics     []string `yaml:"topics"`
	BrokerUrls []*Url   `yaml:"brokerUrls"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	ClientId   string   `yaml:"clientId"`
	KeepAlive  uint16   `yaml:"keepAlive"`
	QoS        byte     `yaml:"qos"`
}

func (m *MqttInput) GetBrokerUrls() []*url.URL {
	urls := make([]*url.URL, len(m.BrokerUrls))
	for i := range m.BrokerUrls {
		urls[i] = m.BrokerUrls[i].URL
	}
	return urls
}
