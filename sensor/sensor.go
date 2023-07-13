package sensor

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
)

type Sensor interface {
	Decode(ctx context.Context, sensorConfig *config.Sensor, data []byte) (*Data, error)
}
