package transform

import (
	"context"
)

type Transform interface {
	Transform(ctx context.Context, data []byte) ([]byte, error)
}
