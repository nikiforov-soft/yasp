package p1p2

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

const (
	RunningBit     = 1 << 0
	ModeCoolingBit = 1 << 3
	ModeHeatingBit = 1 << 6
)

const (
	FanSpeedLowBit    = 1 << 3
	FanSpeedMediumBit = 1 << 2
	FanSpeedHighBit   = 1 << 1
)

type hitachiMessage []byte

func (hm hitachiMessage) running() bool {
	operationMode := hm[0x0A]
	return (operationMode & RunningBit) == RunningBit
}

func (hm hitachiMessage) mode() HitachiMode {
	operationMode := hm[0x0A]
	if (operationMode & ModeCoolingBit) == ModeCoolingBit {
		return HitachiModeCooling
	}
	if (operationMode & ModeHeatingBit) == ModeHeatingBit {
		return HitachiModeHeating
	}
	logrus.WithField("operationMode", operationMode).Error("unknown operation mode")
	return HitachiModeUnknown
}

func (hm hitachiMessage) fanSpeed() HitachiFanSpeed {
	fanSpeed := hm[0x0B]
	if (fanSpeed & FanSpeedLowBit) == FanSpeedLowBit {
		return HitachiFanSpeedLow
	}
	if (fanSpeed & FanSpeedMediumBit) == FanSpeedMediumBit {
		return HitachiFanSpeedMedium
	}
	if (fanSpeed & FanSpeedHighBit) == FanSpeedHighBit {
		return HitachiFanSpeedHigh
	}
	logrus.WithField("fanSpeedMode", fanSpeed).Error("unknown fan speed mode")
	return HitachiFanSpeedUnknown
}

func (hm hitachiMessage) temperature() int {
	return int(hm[0x0C])
}

func (hm hitachiMessage) direction() HitachiDirection {
	direction := HitachiDirection(hm[0])
	switch direction {
	case DirectionUnitsToController,
		DirectionControllerToUnits:
		return direction
	default:
		return DirectionUnknown
	}
}

func (hm hitachiMessage) testMode() bool {
	direction := hm.direction()
	switch direction {
	case DirectionUnitsToController:
		return (hm[0x2A] & 0x08) == 0x08
	case DirectionControllerToUnits:
		return (hm[0x0D] & 0x80) == 0x80
	default:
		logrus.WithField("direction", direction).Error("unknown direction")
		return false
	}
}

func (hm hitachiMessage) errorCode() byte {
	direction := hm.direction()
	switch direction {
	case DirectionUnitsToController:
		return hm[0x0E]
	case DirectionControllerToUnits:
		return 0
	default:
		return 0
	}
}

func (hm hitachiMessage) validate() error {
	direction := hm.direction()
	switch direction {
	case DirectionUnitsToController,
		DirectionControllerToUnits:
	default:
		return fmt.Errorf("%w: %d", ErrUnknownDirection, direction)
	}

	if !hm.validateChecksum() {
		return ErrInvalidChecksum
	}

	return nil
}

func (hm hitachiMessage) validateChecksum() bool {
	var checksum byte
	for _, b := range hm[1 : len(hm)-1] {
		checksum ^= b
	}
	return checksum == hm[len(hm)-1]
}
