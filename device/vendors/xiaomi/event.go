package xiaomi

type Event interface {
	isMiEvent()
	EventType() EventType
}
