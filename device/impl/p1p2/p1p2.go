package p1p2

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
)

const (
	temperatureKey     = "IULiquidPipeTemperature"
	statusKey          = "IUAirInletTemperature"
	fanSpeedKey        = "IUAirOutletTemperature"
	allowedPrefixesKey = "allowedPrefixes"
	skipUnknownKey     = "skipUnknown"

	fanSpeedHighBit   = 1 << 1
	fanSpeedMediumBit = 1 << 2
	fanSpeedLowBit    = 1 << 3
)

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

	if !strings.HasSuffix(inputTopic, temperatureKey) &&
		!strings.HasSuffix(inputTopic, statusKey) &&
		!strings.HasSuffix(inputTopic, fanSpeedKey) {
		if skipUnknown, exists := p.config.Properties[skipUnknownKey]; exists && strings.ToLower(skipUnknown) == "true" {
			return nil, nil
		}
		return data, nil
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

	unit := inputTopic[(lastSlashIndex + 1):]
	value := data.Data

	switch unit {
	case statusKey:
		properties["unit"] = "Status"
		switch string(data.Data) {
		case "8":
			value = []byte("0")
			properties["value"] = float64(0)
			properties["description"] = "Off"
		case "9":
			value = []byte("1")
			properties["value"] = float64(1)
			properties["description"] = "On"
		default:
			logrus.
				WithField("unit", unit).
				WithField("value", string(value)).
				Warn("unsupported value")
			return nil, nil
		}
	case temperatureKey:
		float64Value, err := strconv.ParseFloat(string(data.Data), 64)
		if err != nil {
			return nil, fmt.Errorf("p1p2: failed to parse data: %s as float64: %w", string(data.Data), err)
		}
		properties["unit"] = "Temperature"
		properties["value"] = float64Value
		properties["description"] = "Celsius"
	case fanSpeedKey:
		fanSpeedMask, err := strconv.ParseInt(string(data.Data), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("p1p2: failed to parse data: %s as int: %w", string(data.Data), err)
		}

		var fanSpeed string
		var fanSpeedValue int
		if (fanSpeedMask & fanSpeedMediumBit) == fanSpeedMediumBit {
			fanSpeed = "Medium"
			fanSpeedValue = 2
		} else if (fanSpeedMask & fanSpeedHighBit) == fanSpeedHighBit {
			fanSpeed = "High"
			fanSpeedValue = 3
		} else if (fanSpeedMask & fanSpeedLowBit) == fanSpeedLowBit {
			fanSpeed = "Low"
			fanSpeedValue = 1
		} else {
			logrus.
				WithField("unit", unit).
				WithField("value", fanSpeedMask).
				Warn("unsupported value")
			return nil, nil
		}
		properties["unit"] = "Fan Speed"
		properties["value"] = fanSpeedValue
		properties["description"] = fanSpeed
	default:
		properties["unit"] = unit
		properties["value"] = string(data.Data)
	}

	return &device.Data{
		Data:       value,
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
