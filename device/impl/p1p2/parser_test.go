package p1p2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseHitachiMessage(t *testing.T) {
	tests := []struct {
		name             string
		messageHexString string
		expected         HitachiState
		expectedError    string
	}{
		{
			name:             "test units to remote for fan speed low 22c, power on cooling, test mode off",
			messageHexString: "89002D010A0101010101094816000001140002040000000420010220008088021018131E031000000000000078",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     true,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedLow,
				Temperature: 22,
				TestMode:    false,
			},
			expectedError: "",
		},
		{
			name:             "test units to remote for fan speed high 30c, power on cooling, test mode off",
			messageHexString: "89002D010A010101010109221E000001140002040000000420010220008088021018131E03100000000000001A",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     true,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedHigh,
				Temperature: 30,
				TestMode:    false,
			},
			expectedError: "",
		},
		{
			name:             "test units to remote for fan speed medium 30c, power on cooling, test mode off",
			messageHexString: "89002D010A010101010109241E000001140002040000000420010220008088021018131E03100000000000001C",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     true,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedMedium,
				Temperature: 30,
				TestMode:    false,
			},
			expectedError: "",
		},
		{
			name:             "test units to remote for fan speed low 30c, power on cooling, test mode off",
			messageHexString: "89002D010A010101010109281E000001140002040000000420010220008088021018131E031000000000000010",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     true,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedLow,
				Temperature: 30,
				TestMode:    false,
			},
			expectedError: "",
		},
		{
			name:             "test units to remote for fan speed low 30c, power off cooling, test mode off",
			messageHexString: "89002D010A010101010108281E000001140002040000000420010220008088021018131E031000000000000011",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     false,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedLow,
				Temperature: 30,
				TestMode:    false,
			},
			expectedError: "",
		},
		{
			name:             "test units to remote for fan speed low 30c, power off cooling, test mode on",
			messageHexString: "89002D010A010101010108081E000001140002040000000420010220008088021018131E031000000000080039",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     false,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedLow,
				Temperature: 30,
				TestMode:    true,
			},
			expectedError: "",
		},
		{
			name:             "test units to remote for fan speed high 22c, power on cooling, test mode off",
			messageHexString: "89002D010A0101010101096216000602140000040000000420010220008088021018131E031000000000000055",
			expected: HitachiState{
				Direction:   DirectionUnitsToController,
				Running:     true,
				Mode:        HitachiModeCooling,
				FanMode:     HitachiFanSpeedHigh,
				Temperature: 22,
				TestMode:    false,
				ErrorCode:   6,
			},
			expectedError: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ParseHitachiMessage(test.messageHexString)
			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}
