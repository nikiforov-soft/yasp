package output

import (
	"context"
)

type Output interface {
	Publish(ctx context.Context, data *Data) error
	Close(ctx context.Context) error
}
