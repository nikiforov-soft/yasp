package xiaomi

type EventHumidity struct {
	Humidity float64
}

func (e *EventHumidity) isMiEvent() {}
func (e *EventHumidity) EventType() EventType {
	return EventTypeHumidity
}
