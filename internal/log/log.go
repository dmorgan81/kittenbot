package log

import (
	"context"
	"io"
	"log/slog"

	"github.com/samber/lo"
)

type contextKey struct{}

var discardLogger = New(io.Discard)

func New(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			return lo.Ternary(a.Key == slog.TimeKey, slog.Attr{}, a)
		},
	}))
}

func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContextOrDiscard(ctx context.Context) *slog.Logger {
	if v, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return v
	}
	return discardLogger
}
