package bthome

type SensorData struct {
	CapabilityFlags    CapabilityFlags `json:"capabilityFlags"`
	PacketId           byte            `json:"packetId,omitempty"`
	BatteryPercent     byte            `json:"batteryPercent,omitempty"`
	IlluminanceLux     float32         `json:"illuminanceLux,omitempty"`
	MotionState        byte            `json:"motionState,omitempty"`
	WindowState        byte            `json:"windowState,omitempty"`
	HumidityPercent    byte            `json:"humidityPercent,omitempty"`
	ButtonEvent        uint16          `json:"buttonEvent,omitempty"`
	RotationDegrees    float32         `json:"rotationDegrees,omitempty"`
	TemperatureCelsius float32         `json:"temperatureCelsius,omitempty"`
}
