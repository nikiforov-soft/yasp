package output

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
)

type Factory func(ctx context.Context, config *config.Mqtt) (Output, error)
