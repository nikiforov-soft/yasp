package bletomqtt

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/input/transform"
	"github.com/sirupsen/logrus"
)

const (
	serviceDataKey = "serviceDataKey"
)

type InputEventData struct {
	Id          string            `json:"id"`
	Event       string            `json:"event"`
	MacAddress  string            `json:"mac_address"`
	LocalName   string            `json:"local_name"`
	ServiceData map[string]string `json:"service_data"`
}

type bleToMqtt struct {
	config *config.Transform
}

func (btm *bleToMqtt) Transform(_ context.Context, data []byte) ([]byte, error) {
	var inputEventData *InputEventData
	if err := json.Unmarshal(data, &inputEventData); err != nil {
		return nil, fmt.Errorf("ble-to-mqtt transform: failed to json decode input event data: %w", err)
	}

	logrus.WithField("payload", inputEventData).Debug("input transformed")
	sdk := btm.config.Properties[serviceDataKey]
	if serviceData, exists := inputEventData.ServiceData[sdk]; exists {
		payload, err := hex.DecodeString(serviceData)
		if err != nil {
			return nil, fmt.Errorf("ble-to-mqtt transform: failed to hex decode service data: %w", err)
		}
		return payload, err
	}
	return nil, fmt.Errorf("ble-to-mqtt transform: unable to find %s in service data", sdk)
}

func init() {
	err := transform.RegisterTransform("ble-to-mqtt", func(ctx context.Context, config *config.Transform) (transform.Transform, error) {
		if _, exists := config.Properties[serviceDataKey]; !exists {
			return nil, fmt.Errorf("ble-to-mqtt transform: missing config property: %s", serviceDataKey)
		}
		return &bleToMqtt{
			config: config,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
