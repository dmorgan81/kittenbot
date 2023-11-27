package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmorgan81/kittenbot/internal/handler"
	"github.com/dmorgan81/kittenbot/internal/inject"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
)

func main() {
	ctx := log.NewContext(context.Background(), log.New(os.Stderr))
	ctx, cancel := context.WithCancel(ctx)

	injector := inject.Setup(ctx)

	if _, ok := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); ok {
		handler := do.MustInvoke[*handler.Handler](injector).Handle
		go lambda.StartWithOptions(handler, lambda.WithContext(ctx), lambda.WithEnableSIGTERM(func() {
			cancel()
		}))
	} else {
		var input handler.Input
		if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		handler := do.MustInvoke[*handler.Handler](injector)
		output, err := handler.Handle(ctx, input)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(output)
		cancel()
	}

	select {
	case <-ctx.Done():
		if err := injector.Shutdown(); err != nil {
			fmt.Println(err)
		}
	}
}
