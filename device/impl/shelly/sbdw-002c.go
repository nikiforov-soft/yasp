package shelly

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
)

// https://shelly-api-docs.shelly.cloud/docs-ble/Devices/dw/
func init() {
	err := device.RegisterDevice("SBDW-002C", func(ctx context.Context, config *config.Device) (device.Device, error) {
		return &shellyBtDevice{
			name:       config.Name,
			deviceType: config.Type,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
