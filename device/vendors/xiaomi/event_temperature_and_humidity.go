package xiaomi

type EventTemperatureAndHumidity struct {
	Temperature float64
	Humidity    float64
}

func (e *EventTemperatureAndHumidity) isMiEvent() {}
func (e *EventTemperatureAndHumidity) EventType() EventType {
	return EventTypeTemperatureAndHumidity
}
