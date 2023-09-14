package mqtt

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/memphisdev/memphis.go"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/input"
)

type memphisInput struct {
	config             *config.MemphisInput
	consumers          []*memphis.Consumer
	consumersLock      sync.Mutex
	activeChannels     []chan *input.Data
	activeChannelsLock sync.Mutex
}

func newMemphisInput(_ context.Context, config *config.MemphisInput) (input.Input, error) {
	return &memphisInput{
		config: config,
	}, nil
}

func (mi *memphisInput) Subscribe(ctx context.Context) (<-chan *input.Data, error) {
	conn, err := memphis.Connect(mi.config.Hostname, mi.config.Username, memphis.Password(mi.config.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to memphis: %w", err)
	}

	logrus.Info("memphis input: connected to the server")

	consumer, err := conn.CreateConsumer(mi.config.Station, mi.config.ConsumerName, memphis.PullInterval(mi.config.PollInterval), memphis.BatchSize(mi.config.BatchSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	consumer.SetContext(ctx)
	if err := consumer.Consume(mi.messageHandler); err != nil {
		return nil, err
	}

	mi.consumersLock.Lock()
	mi.consumers = append(mi.consumers, consumer)
	mi.consumersLock.Unlock()

	dataChan := make(chan *input.Data)

	mi.activeChannelsLock.Lock()
	mi.activeChannels = append(mi.activeChannels, dataChan)
	mi.activeChannelsLock.Unlock()

	return dataChan, nil
}

func (mi *memphisInput) Close(_ context.Context) error {
	mi.activeChannelsLock.Lock()
	defer mi.activeChannelsLock.Unlock()

	for _, channel := range mi.activeChannels {
		close(channel)
	}

	mi.consumersLock.Lock()
	defer mi.consumersLock.Unlock()
	var errs []error
	for _, consumer := range mi.consumers {
		if err := consumer.Destroy(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (mi *memphisInput) messageHandler(messages []*memphis.Msg, err error, ctx context.Context) {
	if err != nil {
		logrus.WithError(err).Error("failed to fetch")
		return
	}

	mi.activeChannelsLock.Lock()
	defer mi.activeChannelsLock.Unlock()

	for _, message := range messages {
		logrus.WithField("headers", message.GetHeaders()).WithField("payload", string(message.Data())).Debug("input received")
		data := &input.Data{
			Data:       message.Data(),
			Properties: make(map[string]interface{}),
		}
		for k, v := range message.GetHeaders() {
			data.Properties[k] = v
		}

		if err := message.Ack(); err != nil {
			logrus.WithError(err).Error("failed to ack msg")
			continue
		}

		if mi.config.HeaderPrefixes != nil {
			var exists bool
			for k, v := range message.GetHeaders() {
				if requiredPrefix, exists := mi.config.HeaderPrefixes[k]; exists {
					matches, err := filepath.Match(requiredPrefix, v)
					if err != nil {
						return
					}
					if matches {
						exists = true
						break
					}
				}
			}
			if !exists {
				continue
			}
		}

		for _, channel := range mi.activeChannels {
			channel <- data
		}
	}
}

func init() {
	err := input.RegisterInput("memphis", func(ctx context.Context, config *config.Input) (input.Input, error) {
		return newMemphisInput(ctx, config.Memphis)
	})
	if err != nil {
		panic(err)
	}
}
