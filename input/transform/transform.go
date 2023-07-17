package transform

import (
	"context"

	"github.com/nikiforov-soft/yasp/input"
)

type Transform interface {
	Transform(ctx context.Context, data *input.Data) (*input.Data, error)
}
