package xiaomi

type EventType uint16

const (
	EventTypeTemperature            EventType = 4100
	EventTypeHumidity               EventType = 4102
	EventTypeIlluminance            EventType = 4103
	EventTypeMoisture               EventType = 4104
	EventTypeFertility              EventType = 4105
	EventTypeBattery                EventType = 4106
	EventTypeTemperatureAndHumidity EventType = 4109
)
