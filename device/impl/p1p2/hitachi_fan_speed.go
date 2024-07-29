package p1p2

type HitachiFanSpeed string

func (hfs HitachiFanSpeed) String() string {
	return string(hfs)
}

func (hfs HitachiFanSpeed) Id() int {
	switch hfs {
	case HitachiFanSpeedLow:
		return 1
	case HitachiFanSpeedMedium:
		return 2
	case HitachiFanSpeedHigh:
		return 3
	default:
		return -1
	}
}

const (
	HitachiFanSpeedUnknown HitachiFanSpeed = ""
	HitachiFanSpeedLow     HitachiFanSpeed = "Low"
	HitachiFanSpeedMedium  HitachiFanSpeed = "Medium"
	HitachiFanSpeedHigh    HitachiFanSpeed = "High"
)
