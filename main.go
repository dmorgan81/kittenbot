package main

import (
	"context"
	_ "embed"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmorgan81/kittenbot/internal/handle"
	"github.com/dmorgan81/kittenbot/internal/inject"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/samber/do"
)

func main() {
	log := stdr.New(log.New(os.Stderr, "", 0))
	ctx := logr.NewContext(context.Background(), log)
	injector := inject.Setup(ctx)

	handler := do.MustInvoke[*handle.ImageHandler](injector)
	lambda.StartWithOptions(handler.Handle, lambda.WithContext(ctx), lambda.WithEnableSIGTERM(func() {
		_ = injector.Shutdown()
	}))
}
