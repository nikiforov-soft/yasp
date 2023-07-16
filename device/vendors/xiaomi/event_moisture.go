package xiaomi

type EventMoisture struct {
	Moisture byte
}

func (e *EventMoisture) isMiEvent() {}
func (e *EventMoisture) EventType() EventType {
	return EventTypeMoisture
}
