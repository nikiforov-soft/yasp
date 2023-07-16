package xiaomi

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBLEFrame(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		bindKey     string
		expected    *Frame
		expectedErr string
	}{
		{
			name:        "no bind key",
			data:        "58585b051fa3891338c1a4f30a68073c000000f7058be5",
			bindKey:     "",
			expected:    nil,
			expectedErr: "failed to read encrypted payload: bind key required",
		},
		{
			name:    "humidity event",
			data:    "58585b051fa3891338c1a4f30a68073c000000f7058be5",
			bindKey: "6badc40a09b9176765c76226f000d6cb",
			expected: &Frame{
				FrameControlFlags: FrameControlFlags{
					IsEncrypted:   true,
					HasMacAddress: true,
					HasEvent:      true,
				},
				Version:      5,
				ProductId:    1371,
				FrameCounter: 31,
				Capabilities: CapabilityFlags{},
				MacAddress:   "a4c1381389a3",
				EventType:    4102,
				EventLength:  2,
				Event: &EventHumidity{
					Humidity: 30.8,
				},
			},
			expectedErr: "",
		},
		{
			name:    "temperature event",
			data:    "58585b05a4d8913838c1a4c2e0b504e004000047a2894a",
			bindKey: "e8009ec45eec9e46922c938daf22bfc4",
			expected: &Frame{
				FrameControlFlags: FrameControlFlags{
					IsEncrypted:   true,
					HasMacAddress: true,
					HasEvent:      true,
				},
				Version:      5,
				ProductId:    1371,
				FrameCounter: 164,
				Capabilities: CapabilityFlags{},
				MacAddress:   "a4c1383891d8",
				EventType:    4100,
				EventLength:  2,
				Event: &EventTemperature{
					Temperature: 25.6,
				},
			},
			expectedErr: "",
		},
		{
			name:    "battery event",
			data:    "58585b05a5d8913838c1a4f663838b040000875a963d",
			bindKey: "e8009ec45eec9e46922c938daf22bfc4",
			expected: &Frame{
				FrameControlFlags: FrameControlFlags{
					IsEncrypted:   true,
					HasMacAddress: true,
					HasEvent:      true,
				},
				Version:      5,
				ProductId:    1371,
				FrameCounter: 165,
				Capabilities: CapabilityFlags{},
				MacAddress:   "a4c1383891d8",
				EventType:    4106,
				EventLength:  1,
				Event: &EventBattery{
					Battery: 100,
				},
			},
			expectedErr: "",
		},
		{
			name:    "io event",
			data:    "30585b0558a3891338c1a408",
			bindKey: "6badc40a09b9176765c76226f000d6cb",
			expected: &Frame{
				FrameControlFlags: FrameControlFlags{
					HasMacAddress:   true,
					HasCapabilities: true,
				},
				Version:      5,
				ProductId:    1371,
				FrameCounter: 88,
				Capabilities: CapabilityFlags{
					IO: true,
				},
				MacAddress:  "a4c1381389a3",
				EventType:   0,
				EventLength: 0,
				Event:       nil,
			},
			expectedErr: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rawData, err := hex.DecodeString(test.data)
			assert.NoError(t, err)

			bindKey, err := hex.DecodeString(test.bindKey)
			assert.NoError(t, err)

			actual, err := ParseBLEFrame(rawData, func(mac string) ([]byte, error) {
				return bindKey, nil
			})
			if len(test.expectedErr) != 0 {
				assert.Error(t, err, test.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}
