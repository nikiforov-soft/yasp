package bthome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expected      *SensorData
		expectedError string
	}{
		{
			name:          "invalid length",
			data:          []byte{0x44},
			expected:      nil,
			expectedError: "bthome: invalid data length: 1",
		},
		{
			name:          "invalid header",
			data:          []byte{0x20, 0x00, 0x00},
			expected:      nil,
			expectedError: "bthome: unsupported bthome version: 1",
		},
		{
			name:          "invalid property",
			data:          []byte{0x44, 0x00, 0x00, 0x98, 0x00},
			expected:      nil,
			expectedError: "bthome: unknown property type: 152",
		},
		{
			name: "packet id 43, battery 100%, illuminance 144192.00 lux, window 0, button event 0, rotation 0.0",
			data: []byte{0x44, 0x00, 0x2b, 0x01, 0x64, 0x05, 0xdc, 0x05, 0x00, 0x2d, 0x00, 0x3f, 0x00, 0x00},
			expected: &SensorData{
				CapabilityFlags: CapabilityFlags{
					Encryption:         false,
					TriggerBasedDevice: true,
					Version:            2,
				},
				PacketId:           43,
				BatteryPercent:     100,
				IlluminanceLux:     144192.00,
				MotionState:        0,
				WindowState:        0,
				HumidityPercent:    0,
				ButtonEvent:        0,
				RotationDegrees:    0,
				TemperatureCelsius: 0,
			},
			expectedError: "",
		},
		{
			name: "packet id 44, battery 100%, illuminance 13120.00 lux, window 1, button event 0, rotation 0.0",
			data: []byte{0x44, 0x00, 0x2c, 0x01, 0x64, 0x05, 0x14, 0x05, 0x00, 0x2d, 0x01, 0x3f, 0x00, 0x00},
			expected: &SensorData{
				CapabilityFlags: CapabilityFlags{
					Encryption:         false,
					TriggerBasedDevice: true,
					Version:            2,
				},
				PacketId:        44,
				BatteryPercent:  100,
				IlluminanceLux:  13120.00,
				WindowState:     1,
				ButtonEvent:     0,
				RotationDegrees: 0,
			},
			expectedError: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := Parse(test.data)
			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}
