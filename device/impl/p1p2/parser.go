package p1p2

import (
	"encoding/hex"
)

func ParseHitachiMessage(messageHexString string) (HitachiState, error) {
	bytes, err := hex.DecodeString(messageHexString)
	if err != nil {
		return HitachiState{}, err
	}

	msg := hitachiMessage(bytes)
	if err := msg.validate(); err != nil {
		return HitachiState{}, err
	}

	return HitachiState{
		Direction:   msg.direction(),
		Running:     msg.running(),
		Mode:        msg.mode(),
		FanMode:     msg.fanSpeed(),
		Temperature: msg.temperature(),
		TestMode:    msg.testMode(),
		ErrorCode:   msg.errorCode(),
	}, nil
}
