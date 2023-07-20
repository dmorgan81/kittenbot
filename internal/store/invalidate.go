package store

import (
	"context"
)

type Invalidator interface {
	Invalidate(context.Context, []string) error
}

type NoopInvalidator struct{}

func (*NoopInvalidator) Invalidate(context.Context, []string) error { return nil }
