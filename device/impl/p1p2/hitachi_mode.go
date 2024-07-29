package p1p2

type HitachiMode string

func (hm HitachiMode) String() string {
	return string(hm)
}

func (hm HitachiMode) Id() int {
	switch hm {
	case HitachiModeCooling:
		return 1
	case HitachiModeHeating:
		return 2
	default:
		return -1
	}
}

const (
	HitachiModeUnknown HitachiMode = ""
	HitachiModeCooling HitachiMode = "Cooling"
	HitachiModeHeating HitachiMode = "Heating"
)
