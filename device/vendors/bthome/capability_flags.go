package bthome

type CapabilityFlags struct {
	Encryption         bool `json:"encryption,omitempty"`
	TriggerBasedDevice bool `json:"triggerBasedDevice,omitempty"`
	Version            byte `json:"version,omitempty"`
}
