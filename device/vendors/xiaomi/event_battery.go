package xiaomi

type EventBattery struct {
	Battery byte
}

func (e *EventBattery) isMiEvent() {}
func (e *EventBattery) EventType() EventType {
	return EventTypeBattery
}
