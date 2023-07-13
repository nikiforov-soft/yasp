package lywsd03mmc

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/sensor"
	"github.com/nikiforov-soft/yasp/sensor/vendors/xiaomi"
)

const (
	DeviceType                 = "LYWSD03MMC"
	macAddressPropertiesKey    = "macAddress"
	encryptionKeyPropertiesKey = "encryptionKey"
)

type Sensor struct {
}

func NewSensor() *Sensor {
	return &Sensor{}
}

func (*Sensor) Decode(_ context.Context, sensorConfig *config.Sensor, data []byte) (*sensor.Data, error) {
	var device *config.Device
	frame, err := xiaomi.ParseBLEFrame(data, func(mac string) ([]byte, error) {
		device = findDeviceByMac(sensorConfig.Devices, mac)
		if device == nil {
			return nil, fmt.Errorf("LYWSD03MMC: unable to find bind key for device mac address: %s", mac)
		}

		encryptionKey, err := hex.DecodeString(device.Properties[encryptionKeyPropertiesKey])
		if err != nil {
			return nil, fmt.Errorf("LYWSD03MMC: failed to decode encryption key %w", err)
		}

		return encryptionKey, nil
	})
	if err != nil {
		if !strings.HasPrefix(err.Error(), "LYWSD03MMC") {
			return nil, fmt.Errorf("LYWSD03MMC: %w", err)
		}
		return nil, err
	}

	var processedEvent *ProcessedEvent
	switch event := frame.Event.(type) {
	case *xiaomi.EventTemperature:
		processedEvent = &ProcessedEvent{
			Unit:  "Temperature",
			Value: strconv.FormatFloat(event.Temperature, 'f', 2, 64),
		}
	case *xiaomi.EventHumidity:
		processedEvent = &ProcessedEvent{
			Unit:  "Humidity",
			Value: strconv.FormatFloat(event.Humidity, 'f', 2, 64),
		}
	case *xiaomi.EventBattery:
		processedEvent = &ProcessedEvent{
			Unit:  "Battery",
			Value: strconv.FormatInt(int64(event.Battery), 10),
		}
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("LYWSD03MMC: unhandled sensor data: device name: %s, macAddress: %s, event: %T", device.Name, frame.MacAddress, event)
	}

	return &sensor.Data{
		Data: []byte(processedEvent.Value),
		Properties: map[string]interface{}{
			"deviceName":       device.Name,
			"deviceType":       device.Type,
			"deviceMacAddress": device.Properties[macAddressPropertiesKey],
			"unit":             processedEvent.Unit,
			"value":            processedEvent.Value,
		},
	}, nil
}

func findDeviceByMac(devices []*config.Device, mac string) *config.Device {
	for _, device := range devices {
		if device.Type != DeviceType {
			continue
		}

		macAddress, exists := device.Properties[macAddressPropertiesKey]
		if !exists {
			continue
		}

		if !strings.EqualFold(mac, macAddress) {
			continue
		}

		_, exists = device.Properties[encryptionKeyPropertiesKey]
		if !exists {
			continue
		}

		return device
	}
	return nil
}

func init() {
	err := sensor.RegisterSensor(DeviceType, func(ctx context.Context, config *config.Device) (sensor.Sensor, error) {
		return NewSensor(), nil
	})
	if err != nil {
		panic(err)
	}
}
