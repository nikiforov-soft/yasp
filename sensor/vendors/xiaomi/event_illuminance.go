package xiaomi

type EventIlluminance struct {
	Illuminance uint
}

func (e *EventIlluminance) isMiEvent() {}
func (e *EventIlluminance) EventType() EventType {
	return EventTypeIlluminance
}
