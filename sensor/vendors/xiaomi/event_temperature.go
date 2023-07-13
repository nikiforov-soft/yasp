package xiaomi

type EventTemperature struct {
	Temperature float64
}

func (e *EventTemperature) isMiEvent() {}
func (e *EventTemperature) EventType() EventType {
	return EventTypeTemperature
}
