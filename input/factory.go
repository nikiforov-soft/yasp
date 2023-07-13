package input

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
)

type Factory func(ctx context.Context, config *config.Mqtt) (Input, error)
