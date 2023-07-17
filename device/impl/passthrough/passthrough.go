package passthrough

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
)

type passthrough struct {
	config *config.Device
}

func (p *passthrough) Decode(_ context.Context, data *device.Data) (*device.Data, error) {
	properties := make(map[string]interface{}, len(data.Properties)+4)
	for k, v := range data.Properties {
		properties[k] = v
	}
	properties["deviceName"] = p.config.Name
	properties["deviceType"] = p.config.Type
	properties["deviceProperties"] = p.config.Properties
	properties["value"] = string(data.Data)
	return &device.Data{
		Data:       data.Data,
		Properties: properties,
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
