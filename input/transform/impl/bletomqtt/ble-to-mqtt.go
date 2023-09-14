package bletomqtt

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/input"
	"github.com/nikiforov-soft/yasp/input/transform"
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
	sdk    string
}

func (btm *bleToMqtt) Transform(_ context.Context, data *input.Data) (*input.Data, error) {
	var inputEventData *InputEventData
	if err := json.Unmarshal(data.Data, &inputEventData); err != nil {
		return nil, fmt.Errorf("ble-to-mqtt transform: failed to json decode input event data: %w", err)
	}

	logrus.WithField("payload", inputEventData).Debug("input transformed")
	if serviceData, exists := inputEventData.ServiceData[btm.sdk]; exists {
		payload, err := hex.DecodeString(serviceData)
		if err != nil {
			return nil, fmt.Errorf("ble-to-mqtt transform: failed to hex decode service data: %w", err)
		}

		properties := make(map[string]interface{}, len(data.Properties)+4)
		for k, v := range data.Properties {
			properties[k] = v
		}
		properties["bleToMqttId"] = inputEventData.Id
		properties["bleToMqttEvent"] = inputEventData.Event
		properties["bleToMqttMacAddress"] = inputEventData.MacAddress
		properties["bleToMqttLocalName"] = inputEventData.LocalName
		return &input.Data{
			Data:       payload,
			Properties: properties,
		}, err
	}
	return nil, fmt.Errorf("ble-to-mqtt transform: unable to find %s in service data", btm.sdk)
}

func init() {
	err := transform.RegisterTransform("ble-to-mqtt", func(ctx context.Context, config *config.Transform) (transform.Transform, error) {
		sdk, exists := config.Properties[serviceDataKey]
		if !exists {
			return nil, fmt.Errorf("ble-to-mqtt transform: missing config property: %s", serviceDataKey)
		}
		return &bleToMqtt{
			config: config,
			sdk:    sdk,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
