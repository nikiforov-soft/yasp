package xiaomi

const (
	capabilityBitConnectable = 1 << 0
	capabilityBitCentral     = 1 << 1
	capabilityBitSecure      = 1 << 2
	capabilityBitIO          = (1 << 3) | (1 << 4)
)
