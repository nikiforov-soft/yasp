package mqtt

import (
	"context"
	"fmt"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/metrics"
	"github.com/nikiforov-soft/yasp/output"
	"github.com/nikiforov-soft/yasp/template"
)

var (
	eventsProcessedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "output_events_published",
		Help:      "The amount of events mqtt output published.",
		Namespace: "yasp",
		Subsystem: "mqtt",
	}, []string{"topic"})
)

type mqttOutput struct {
	config            *config.MqttOutput
	connectionManager *autopaho.ConnectionManager
}

func NewMqttOutput(ctx context.Context, config *config.MqttOutput) (output.Output, error) {
	keepAlive := config.KeepAlive
	if keepAlive == 0 {
		keepAlive = 5
	}

	clientConfig := autopaho.ClientConfig{
		BrokerUrls:       config.GetBrokerUrls(),
		KeepAlive:        keepAlive,
		ReconnectBackoff: autopaho.DefaultExponentialBackoff(),
		OnConnectionUp:   func(*autopaho.ConnectionManager, *paho.Connack) { logrus.Info("mqtt output: connected to server") },
		OnConnectError:   func(err error) { logrus.WithError(err).Error("mqtt output: failed to connect to  server") },
		ClientConfig: paho.ClientConfig{
			ClientID:      config.ClientId,
			OnClientError: func(err error) { logrus.WithError(err).Error("output mqtt server requested disconnect") },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					logrus.WithField("reason", d.Properties.ReasonString).Error("mqtt output: server requested disconnect")
				} else {
					logrus.WithField("reasonCode", d.ReasonCode).Error("mqtt output: server requested disconnect")
				}
			},
		},
	}

	if len(config.Username) != 0 && len(config.Password) != 0 {
		clientConfig.ConnectUsername = config.Username
		clientConfig.ConnectPassword = []byte(config.Password)
	}

	connectionManager, err := autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("mqtt output: failed to initialize connection manager: %w", err)
	}

	return &mqttOutput{
		config:            config,
		connectionManager: connectionManager,
	}, nil
}

func (mo *mqttOutput) Publish(ctx context.Context, data *output.Data) error {
	if err := mo.connectionManager.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("mqtt output: failed to await connection: %w", err)
	}

	topicData, err := template.Execute("mqtt output", mo.config.Topic, data)
	if err != nil {
		return fmt.Errorf("mqtt output: failed to parse glob: %w", err)
	}

	topic := string(topicData)
	logrus.WithField("topic", topic).WithField("payload", string(data.Data)).Debug("output published")
	_, err = mo.connectionManager.Publish(ctx, &paho.Publish{
		QoS:     mo.config.QoS,
		Retain:  mo.config.Retain,
		Topic:   topic,
		Payload: data.Data,
	})
	if err != nil {
		return fmt.Errorf("mqtt output: failed to publish message: %w", err)
	}

	eventsProcessedCounter.WithLabelValues(topic).Inc()

	return nil
}

func (mo *mqttOutput) Close(ctx context.Context) error {
	return mo.connectionManager.Disconnect(ctx)
}

func init() {
	err := output.RegisterOutput("mqtt", func(ctx context.Context, config *config.Output, metricsService metrics.Service) (output.Output, error) {
		return NewMqttOutput(ctx, config.Mqtt)
	})
	if err != nil {
		panic(err)
	}
}
