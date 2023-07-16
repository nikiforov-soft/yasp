package xiaomi

type EventFertility struct {
	Fertility int16
}

func (e *EventFertility) isMiEvent() {}
func (e *EventFertility) EventType() EventType {
	return EventTypeFertility
}
