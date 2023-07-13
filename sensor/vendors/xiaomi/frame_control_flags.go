package xiaomi

type FrameControlFlags struct {
	IsFactoryNew    bool
	IsConnected     bool
	IsCentral       bool
	IsEncrypted     bool
	HasMacAddress   bool
	HasCapabilities bool
	HasEvent        bool
	HasCustomData   bool
	HasSubtitle     bool
	HasBinding      bool
}
