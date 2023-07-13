package xiaomi

type Frame struct {
	FrameControlFlags FrameControlFlags
	Version           uint8
	ProductId         uint16
	FrameCounter      uint8
	Capabilities      CapabilityFlags
	MacAddress        string
	EventType         uint16
	EventLength       uint8
	Event             Event
}
