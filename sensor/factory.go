package sensor

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
)

type Factory func(ctx context.Context, config *config.Device) (Sensor, error)
