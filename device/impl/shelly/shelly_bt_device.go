package shelly

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/nikiforov-soft/yasp/device"
	"github.com/nikiforov-soft/yasp/device/vendors/bthome"
)

type shellyBtDevice struct {
	name       string
	deviceType string
}

func (sbd *shellyBtDevice) Decode(_ context.Context, data *device.Data) (*device.Data, error) {
	sensorData, err := bthome.Parse(data.Data)
	if err != nil {
		return nil, fmt.Errorf("shelly: failed to parse sensor data: %s - %w", hex.EncodeToString(data.Data), err)
	}

	result, err := json.Marshal(sensorData)
	if err != nil {
		return nil, fmt.Errorf("shelly: failed to marshal sensor data: %w", err)
	}

	properties := make(map[string]interface{}, len(data.Properties)+8)
	for k, v := range data.Properties {
		properties[k] = v
	}
	properties["deviceName"] = sbd.name
	properties["deviceType"] = sbd.deviceType
	properties["batteryPercent"] = sensorData.BatteryPercent
	properties["illuminanceLux"] = sensorData.IlluminanceLux
	properties["motionState"] = sensorData.MotionState
	properties["windowState"] = sensorData.WindowState
	properties["humidityPercent"] = sensorData.HumidityPercent
	properties["buttonEvent"] = sensorData.ButtonEvent
	properties["rotationDegrees"] = sensorData.RotationDegrees
	properties["temperatureCelsius"] = sensorData.TemperatureCelsius

	return &device.Data{
		Data:       result,
		Properties: properties,
	}, nil
}
