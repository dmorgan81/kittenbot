package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	path := filepath.Base(os.Args[0])
	var handler any
	switch path {
	case "kittenbot-image":
		handler = do.MustInvoke[*handle.ImageHandler](injector).Handle
	case "kittenbot-html":
		handler = do.MustInvoke[*handle.HtmlHandler](injector).Handle
	default:
		fmt.Printf("no such handler: %s\n", path)
		os.Exit(1)
	}

	lambda.StartWithOptions(handler, lambda.WithContext(ctx), lambda.WithEnableSIGTERM(func() {
		_ = injector.Shutdown()
	}))
}
