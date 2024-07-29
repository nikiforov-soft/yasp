package p1p2

type HitachiState struct {
	Direction   HitachiDirection
	Running     bool
	Mode        HitachiMode
	FanMode     HitachiFanSpeed
	Temperature int
	TestMode    bool
	ErrorCode   byte
}
