package mqtt

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/output"
	"github.com/sirupsen/logrus"
)

var (
	funcsMap = map[string]any{
		"ToLower":    strings.ToLower,
		"ToUpper":    strings.ToUpper,
		"TrimSpaces": strings.TrimSpace,
		"TrimPrefix": strings.TrimPrefix,
		"TrimSuffix": strings.TrimSuffix,
	}
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

	connectRetryDelay := config.ConnectRetryDelay
	if connectRetryDelay == 0 {
		connectRetryDelay = 5 * time.Second
	}

	clientConfig := autopaho.ClientConfig{
		BrokerUrls:        config.GetBrokerUrls(),
		KeepAlive:         keepAlive,
		ConnectRetryDelay: connectRetryDelay,
		OnConnectionUp:    func(*autopaho.ConnectionManager, *paho.Connack) { logrus.Info("mqtt output: connected to server") },
		OnConnectError:    func(err error) { logrus.WithError(err).Error("mqtt output: failed to connect to  server") },
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

	tmpl, err := template.New("mqtt-output").Funcs(funcsMap).Parse(mo.config.Topic)
	if err != nil {
		return fmt.Errorf("mqtt output: failed to parse glob: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("mqtt output: failed to execute output topic template: %w", err)
	}

	topic := buf.String()
	logrus.WithField("topic", topic).WithField("payload", string(data.Data)).Debug("output published")
	_, err = mo.connectionManager.Publish(ctx, &paho.Publish{
		QoS:     0,
		Topic:   topic,
		Payload: data.Data,
	})
	if err != nil {
		return fmt.Errorf("mqtt output: failed to publish message: %w", err)
	}

	return nil
}

func (mo *mqttOutput) Close(ctx context.Context) error {
	return mo.connectionManager.Disconnect(ctx)
}

func init() {
	err := output.RegisterOutput("mqtt", func(ctx context.Context, config *config.Output) (output.Output, error) {
		return NewMqttOutput(ctx, config.Mqtt)
	})
	if err != nil {
		panic(err)
	}
}
