package input

import (
	"context"
)

type Input interface {
	Subscribe(ctx context.Context) (<-chan *Data, error)
	Close(ctx context.Context) error
}
