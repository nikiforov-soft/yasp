package output

import (
	"context"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/metrics"
)

type Factory func(ctx context.Context, config *config.Output, metricsService metrics.Service) (Output, error)
