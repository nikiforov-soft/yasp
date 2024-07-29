package p1p2

import (
	"errors"
)

var (
	ErrUnknownDirection = errors.New("unknown direction")
	ErrInvalidChecksum  = errors.New("invalid checksum")
)
