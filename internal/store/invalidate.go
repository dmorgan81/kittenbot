package store

import (
	"context"
)

type Invalidator interface {
	Invalidate(context.Context, []string) error
}
