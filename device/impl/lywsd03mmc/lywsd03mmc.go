package lywsd03mmc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
	"github.com/nikiforov-soft/yasp/device/vendors/xiaomi"
)

const (
	deviceType                 = "LYWSD03MMC"
	macAddressPropertiesKey    = "macAddress"
	encryptionKeyPropertiesKey = "encryptionKey"
)

type lywsd03mmc struct {
	name          string
	macAddress    string
	encryptionKey []byte
}

func (s *lywsd03mmc) Decode(_ context.Context, data []byte) (*device.Data, error) {
	frame, err := xiaomi.ParseBLEFrame(data, func(mac string) ([]byte, error) {
		if !strings.EqualFold(s.macAddress, mac) {
			return nil, nil
		}
		return s.encryptionKey, nil
	})
	if err != nil {
		if errors.Is(err, xiaomi.ErrBindKeyRequired) {
			return nil, nil
		}
		if !strings.HasPrefix(err.Error(), "LYWSD03MMC") {
			return nil, fmt.Errorf("LYWSD03MMC: %w", err)
		}
		return nil, err
	}

	var unit string
	var value string
	switch event := frame.Event.(type) {
	case *xiaomi.EventTemperature:
		unit = "Temperature"
		value = strconv.FormatFloat(event.Temperature, 'f', 2, 64)
	case *xiaomi.EventHumidity:
		unit = "Humidity"
		value = strconv.FormatFloat(event.Humidity, 'f', 2, 64)
	case *xiaomi.EventBattery:
		unit = "Battery"
		value = strconv.FormatInt(int64(event.Battery), 10)
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("LYWSD03MMC: unhandled sensor data: device name: %s, macAddress: %s, event: %T", s.name, frame.MacAddress, event)
	}

	return &device.Data{
		Data: []byte(value),
		Properties: map[string]interface{}{
			"deviceName":       s.name,
			"deviceType":       deviceType,
			"deviceMacAddress": s.macAddress,
			"unit":             unit,
			"value":            value,
		},
	}, nil
}

func init() {
	err := device.RegisterDevice(deviceType, func(ctx context.Context, config *config.Device) (device.Device, error) {
		macAddressValue, exists := config.Properties[macAddressPropertiesKey]
		if !exists {
			return nil, fmt.Errorf("LYWSD03MMC: device %s is missing %s property", config.Name, macAddressPropertiesKey)
		}

		macAddress, err := net.ParseMAC(macAddressValue)
		if err != nil {
			return nil, fmt.Errorf("LYWSD03MMC: device %s has invalid mac address value: %w", config.Name, err)
		}

		encryptionKeyValue, exists := config.Properties[encryptionKeyPropertiesKey]
		if !exists {
			return nil, fmt.Errorf("LYWSD03MMC: device %s is missing %s property", config.Name, encryptionKeyPropertiesKey)
		}

		encryptionKey, err := hex.DecodeString(encryptionKeyValue)
		if err != nil {
			return nil, fmt.Errorf("LYWSD03MMC: device %s has invalid encryption key value: %w", config.Name, err)
		}

		return &lywsd03mmc{
			name:          config.Name,
			macAddress:    macAddress.String(),
			encryptionKey: encryptionKey,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
