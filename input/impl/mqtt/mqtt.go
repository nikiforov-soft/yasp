package mqtt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/input"
)

var (
	eventsProcessedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "input_events_processed",
		Help:      "The amount of events mqtt input processed.",
		Namespace: "yasp",
		Subsystem: "mqtt",
	}, []string{"topic"})
)

type mqttInput struct {
	config                 *config.MqttInput
	connectionManagers     []*autopaho.ConnectionManager
	connectionManagersLock sync.Mutex
	activeChannels         []chan *input.Data
	activeChannelsLock     sync.Mutex
}

func newMqttInput(_ context.Context, config *config.MqttInput) (input.Input, error) {
	return &mqttInput{
		config: config,
	}, nil
}

func (mi *mqttInput) Subscribe(ctx context.Context) (<-chan *input.Data, error) {
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

			subscriptions := make([]paho.SubscribeOptions, 0, len(mi.config.Topics))
			for _, topic := range mi.config.Topics {
				subscriptions = append(subscriptions, paho.SubscribeOptions{
					Topic: topic,
					QoS:   0,
				})
			}

			if _, err := cm.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: subscriptions,
			}); err != nil {
				logrus.
					WithError(err).
					Error("mqtt input: failed to subscribe")
				return
			}
			logrus.
				WithField("topics", mi.config.Topics).
				Info("mqtt input: mqtt subscribed")
		},
		OnConnectError: func(err error) { logrus.WithError(err).Error("mqtt input: failed to connect to server") },
		ClientConfig: paho.ClientConfig{
			Router:        paho.NewStandardRouterWithDefault(mi.messageHandler),
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

	if len(mi.config.Username) != 0 && len(mi.config.Password) != 0 {
		clientConfig.ConnectUsername = mi.config.Username
		clientConfig.ConnectPassword = []byte(mi.config.Password)
	}

	mi.activeChannelsLock.Lock()
	defer mi.activeChannelsLock.Unlock()

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

	eventsProcessedCounter.WithLabelValues(publish.Topic).Inc()

	logrus.WithField("payload", string(publish.Payload)).Debug("input received")
	for _, channel := range mi.activeChannels {
		channel <- &input.Data{
			Data: publish.Payload,
			Properties: map[string]interface{}{
				"inputId":         publish.PacketID,
				"inputQos":        publish.QoS,
				"inputRetain":     publish.Retain,
				"inputTopic":      publish.Topic,
				"inputProperties": publish.Properties,
			},
		}
	}
}

func init() {
	err := input.RegisterInput("mqtt", func(ctx context.Context, config *config.Input) (input.Input, error) {
		return newMqttInput(ctx, config.Mqtt)
	})
	if err != nil {
		panic(err)
	}
}
