package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmorgan81/kittenbot/internal/handler"
	"github.com/dmorgan81/kittenbot/internal/inject"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
)

func main() {
	ctx := log.NewContext(context.Background(), log.New(os.Stderr))
	injector := inject.Setup(ctx)
	handler := do.MustInvoke[*handler.Handler](injector)
	lambda.StartWithOptions(handler.Handle, lambda.WithContext(ctx), lambda.WithEnableSIGTERM(func() {
		_ = injector.Shutdown()
	}))
}
