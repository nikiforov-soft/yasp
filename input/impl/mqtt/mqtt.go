package mqtt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/input"
	"github.com/sirupsen/logrus"
)

type mqttInput struct {
	config                 *config.Mqtt
	connectionManagers     []*autopaho.ConnectionManager
	connectionManagersLock sync.Mutex
	activeChannels         []chan *input.Data
	activeChannelsLock     sync.Mutex
}

func NewMqttInput(_ context.Context, config *config.Mqtt) (input.Input, error) {
	return &mqttInput{
		config: config,
	}, nil
}

func (mi *mqttInput) Subscribe(ctx context.Context) (chan *input.Data, error) {
	mi.activeChannelsLock.Lock()
	defer mi.activeChannelsLock.Unlock()

	keepAlive := mi.config.KeepAlive
	if keepAlive == 0 {
		keepAlive = 5
	}

	connectRetryDelay := mi.config.ConnectRetryDelay
	if connectRetryDelay == 0 {
		connectRetryDelay = 5 * time.Second
	}
	clientConfig := autopaho.ClientConfig{
		BrokerUrls:        mi.config.GetBrokerUrls(),
		KeepAlive:         keepAlive,
		ConnectRetryDelay: connectRetryDelay,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			logrus.Info("mqtt input: connected to mqtt server")
			if _, err := cm.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: map[string]paho.SubscribeOptions{
					mi.config.Topic: {QoS: 0},
				},
			}); err != nil {
				logrus.
					WithError(err).
					Error("mqtt input: failed to subscribe")
				return
			}
			logrus.
				WithField("topic", mi.config.Topic).
				Info("mqtt input: mqtt subscribed")
		},
		OnConnectError: func(err error) { logrus.WithError(err).Error("mqtt input: failed to connect to server") },
		ClientConfig: paho.ClientConfig{
			Router:        paho.NewSingleHandlerRouter(mi.messageHandler),
			OnClientError: func(err error) { logrus.WithError(err).Error("mqtt input: server requested disconnect") },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					logrus.WithField("reason", d.Properties.ReasonString).Error("mqtt input: server requested disconnect")
				} else {
					logrus.WithField("reasonCode", d.ReasonCode).Error("mqtt input: server requested disconnect")
				}
			},
		},
	}

	connectionManager, err := autopaho.NewConnection(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("mqtt input: failed to initialize connection manager: %w", err)
	}
	mi.connectionManagers = append(mi.connectionManagers, connectionManager)

	dataChan := make(chan *input.Data)
	mi.activeChannels = append(mi.activeChannels, dataChan)
	return dataChan, nil
}

func (mi *mqttInput) Close(ctx context.Context) error {
	mi.activeChannelsLock.Lock()
	defer mi.activeChannelsLock.Unlock()

	for _, channel := range mi.activeChannels {
		close(channel)
	}

	mi.connectionManagersLock.Lock()
	defer mi.connectionManagersLock.Unlock()
	var errs []error
	for _, connectionManager := range mi.connectionManagers {
		if err := connectionManager.Disconnect(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (mi *mqttInput) messageHandler(publish *paho.Publish) {
	mi.activeChannelsLock.Lock()
	defer mi.activeChannelsLock.Unlock()

	logrus.WithField("payload", string(publish.Payload)).Debug("input received")
	for _, channel := range mi.activeChannels {
		channel <- &input.Data{
			Data: publish.Payload,
			Properties: map[string]interface{}{
				"id":         publish.PacketID,
				"qos":        publish.QoS,
				"retain":     publish.Retain,
				"topic":      publish.Topic,
				"properties": publish.Properties,
			},
		}
	}
}

func init() {
	err := input.RegisterInput("mqtt", func(ctx context.Context, config *config.Mqtt) (input.Input, error) {
		return NewMqttInput(ctx, config)
	})
	if err != nil {
		panic(err)
	}
}
