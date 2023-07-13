package xiaomi

const (
	frameControlBitIsFactoryNew    = 1 << 0
	frameControlBitIsConnected     = 1 << 1
	frameControlBitIsCentral       = 1 << 2
	frameControlBitIsEncrypted     = 1 << 3
	frameControlBitHasMacAddress   = 1 << 4
	frameControlBitHasCapabilities = 1 << 5
	frameControlBitHasEvent        = 1 << 6
	frameControlBitHasCustomData   = 1 << 7
	frameControlBitHasSubtitle     = 1 << 8
	frameControlBitHasBinding      = 1 << 9
)
