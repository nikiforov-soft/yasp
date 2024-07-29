package p1p2

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
)

const allowedPrefixesKey = "allowedPrefixes"

var p1p2MessagePattern = regexp.MustCompile("^[a-zA-Z] [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [a-zA-Z] +([0-9.]+:)? ([a-fA-F0-9]+)$")

type p1p2 struct {
	config *config.Device
}

func (p *p1p2) Decode(_ context.Context, data *device.Data) (*device.Data, error) {
	inputTopic, ok := data.Properties["inputTopic"].(string)
	if !ok {
		return data, nil
	}
	inputTopic = strings.TrimSpace(inputTopic)

	if prefixes, ok := p.config.Properties[allowedPrefixesKey]; ok {
		var matchesPrefix bool
		for _, prefix := range strings.Split(prefixes, ",") {
			if strings.HasPrefix(inputTopic, strings.TrimSpace(prefix)) {
				matchesPrefix = true
				break
			}
		}
		if !matchesPrefix {
			return nil, nil
		}
	}

	lastSlashIndex := strings.LastIndex(inputTopic, "/")
	if lastSlashIndex == -1 {
		return data, nil
	}

	if len(inputTopic) < (lastSlashIndex + 1) {
		return data, nil
	}

	properties := make(map[string]interface{}, len(data.Properties))
	for k, v := range data.Properties {
		properties[k] = v
	}
	properties["type"] = "p1p2"
	properties["bridge"] = inputTopic[lastSlashIndex+1:]

	stringPayload := string(data.Data)
	payloadMatches := p1p2MessagePattern.FindStringSubmatch(stringPayload)
	if !strings.HasPrefix(stringPayload, "R ") {
		return nil, nil
	}

	if len(payloadMatches) != 3 {
		return nil, fmt.Errorf("invalid payload message format: %s", stringPayload)
	}

	hexString := strings.TrimSpace(payloadMatches[2])
	if len(hexString) == 4 {
		return nil, nil
	}

	msg, err := ParseHitachiMessage(hexString)
	if err != nil {
		if errors.Is(err, ErrUnknownDirection) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse p1p2 packet: %s - %w", stringPayload, err)
	}

	properties["temperature"] = strconv.Itoa(msg.Temperature)
	properties["mode"] = msg.Mode.String()
	properties["modeId"] = msg.Mode.Id()
	properties["fanSpeed"] = msg.FanMode.String()
	properties["fanSpeedId"] = msg.FanMode.Id()
	properties["status"] = strconv.FormatBool(msg.Running)
	properties["testMode"] = strconv.FormatBool(msg.TestMode)
	properties["errorCode"] = strconv.Itoa(int(msg.ErrorCode))

	return &device.Data{
		Data:       data.Data,
		Properties: properties,
	}, nil
}

func init() {
	err := device.RegisterDevice("p1p2", func(ctx context.Context, config *config.Device) (device.Device, error) {
		return &p1p2{
			config: config,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
