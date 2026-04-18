package contracts

import "context"

type Store interface {
	Push(ctx context.Context, logs []interface{}) error
}
