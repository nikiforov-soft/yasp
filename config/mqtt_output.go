package config

import (
	"net/url"
	"time"
)

type MqttOutput struct {
	Enabled           bool          `yaml:"enabled"`
	Topic             string        `yaml:"topic"`
	BrokerUrls        []*Url        `yaml:"brokerUrls"`
	ClientId          string        `yaml:"clientId"`
	KeepAlive         uint16        `yaml:"keepAlive"`
	ConnectRetryDelay time.Duration `yaml:"connectRetryDelay"`
}

func (m *MqttOutput) GetBrokerUrls() []*url.URL {
	urls := make([]*url.URL, len(m.BrokerUrls))
	for i := range m.BrokerUrls {
		urls[i] = m.BrokerUrls[i].URL
	}
	return urls
}
