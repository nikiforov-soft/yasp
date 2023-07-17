package transform

import (
	"context"

	"github.com/nikiforov-soft/yasp/output"
)

type Transform interface {
	Transform(ctx context.Context, data *output.Data) (*output.Data, error)
}
