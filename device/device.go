package device

import (
	"context"
)

type Device interface {
	Decode(ctx context.Context, data *Data) (*Data, error)
}
