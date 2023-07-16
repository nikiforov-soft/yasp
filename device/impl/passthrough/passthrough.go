package passthrough

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
)

type passthrough struct {
	config *config.Device
}

func (p *passthrough) Decode(_ context.Context, data []byte) (*device.Data, error) {
	return &device.Data{
		Data: data,
		Properties: map[string]interface{}{
			"deviceName":       p.config.Name,
			"deviceType":       p.config.Type,
			"deviceProperties": p.config.Properties,
		},
	}, nil
}

func init() {
	err := device.RegisterDevice("passthrough", func(ctx context.Context, config *config.Device) (device.Device, error) {
		return &passthrough{config: config}, nil
	})
	if err != nil {
		panic(err)
	}
}
