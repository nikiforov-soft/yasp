package p1p2

type HitachiDirection byte

const (
	DirectionUnknown           HitachiDirection = 0
	DirectionControllerToUnits HitachiDirection = 0x21
	DirectionUnitsToController HitachiDirection = 0x89
)
