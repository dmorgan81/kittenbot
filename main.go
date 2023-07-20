package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/dmorgan81/kittenbot/internal/image"
	"github.com/dmorgan81/kittenbot/internal/page"
	"github.com/dmorgan81/kittenbot/internal/param"
	"github.com/dmorgan81/kittenbot/internal/prompt"
	"github.com/dmorgan81/kittenbot/internal/store"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type Params struct {
	Id     string `json:"id,omitempty"`
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Seed   string `json:"seed,omitempty"`
}

func (p Params) toImageParams() image.Params {
	return image.Params{
		Model:  p.Model,
		Prompt: p.Prompt,
		Seed:   p.Seed,
	}
}

func (p Params) toPageParams() page.Params {
	return page.Params{
		Image:  p.Id + ".png",
		Model:  p.Model,
		Prompt: p.Prompt,
		Seed:   p.Seed,
	}
}

func (p Params) toMetadata() map[string]string {
	return map[string]string{
		"Id":     p.Id,
		"Model":  p.Model,
		"Prompt": p.Prompt,
		"Seed":   p.Seed,
	}
}

type Handler struct {
	Randomizer  *prompt.Randomizer
	Generator   image.Generator
	Templator   *page.Templator
	Uploader    store.Uploader
	Invalidator store.Invalidator
}

func (h *Handler) HandleRequest(ctx context.Context, params Params) (Params, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("handler").WithValues("params", params)
	log.Info("handling lambda invocation")

	if params.Model == "" || params.Prompt == "" {
		model, prompt, err := h.Randomizer.Randomize(ctx)
		if err != nil {
			return params, err
		}
		params.Model = lo.Ternary(params.Model != "", params.Model, model)
		params.Prompt = lo.Ternary(params.Prompt != "", params.Prompt, prompt)
	}

	if params.Id == "" {
		params.Id = time.Now().UTC().Format("20060102")
	}

	img, seed, err := h.Generator.Generate(ctx, params.toImageParams())
	if err != nil {
		return params, err
	}
	params.Seed = seed

	html, err := h.Templator.Template(ctx, params.toPageParams())
	if err != nil {
		return params, err
	}

	metadata := params.toMetadata()
	uploads := []store.UploadParams{
		{Name: params.Id + ".png", Data: img, ContentType: "image/png", Metadata: metadata},
		{Name: "latest.png", Data: img, ContentType: "image/png", Metadata: metadata},
		{Name: params.Id + ".html", Data: html, ContentType: "text/html", Metadata: metadata},
		{Name: "latest.html", Data: html, ContentType: "text/html", Metadata: metadata},
	}
	for _, u := range uploads {
		if err := h.Uploader.Upload(ctx, u); err != nil {
			return params, err
		}
	}

	paths := lo.Map(uploads, func(u store.UploadParams, _ int) string { return "/" + u.Name })
	if err := h.Invalidator.Invalidate(ctx, paths); err != nil {
		return params, err
	}

	return params, nil
}

func main() {
	log := stdr.New(log.New(os.Stderr, "", 0))
	ctx := logr.NewContext(context.Background(), log)

	injector := do.NewWithOpts(&do.InjectorOpts{
		Logf: func(format string, args ...any) {
			log.Info(fmt.Sprintf(format, args))
		},
	})
	do.Provide[aws.Config](injector, func(i *do.Injector) (aws.Config, error) {
		return config.LoadDefaultConfig(ctx)
	})
	do.Provide[*ssm.Client](injector, func(i *do.Injector) (*ssm.Client, error) {
		return ssm.NewFromConfig(do.MustInvoke[aws.Config](i)), nil
	})
	do.Provide[*s3.Client](injector, func(i *do.Injector) (*s3.Client, error) {
		return s3.NewFromConfig(do.MustInvoke[aws.Config](i)), nil
	})
	do.Provide[*cloudfront.Client](injector, func(i *do.Injector) (*cloudfront.Client, error) {
		return cloudfront.NewFromConfig(do.MustInvoke[aws.Config](i)), nil
	})
	do.ProvideValue[*http.Client](injector, http.DefaultClient)

	do.Provide[param.Fetcher](injector, param.NewParameterStoreFetcher)
	do.Provide[*prompt.Randomizer](injector, prompt.NewRandomizer)
	do.Provide[image.Generator](injector, image.NewDezgoGenerator)
	do.Provide[store.Uploader](injector, store.NewS3Uploader)
	do.Provide[store.Invalidator](injector, store.NewCloudFrontInvalidator)
	do.Provide[*page.Templator](injector, page.NewTemplator)

	do.ProvideNamed[string](injector, "dezgo_key", func(i *do.Injector) (string, error) {
		return do.MustInvoke[param.Fetcher](i).Fetch(ctx, os.Getenv("DEZGO_KEY_PARAM"))
	})
	do.ProvideNamed[[]string](injector, "prompts", func(i *do.Injector) ([]string, error) {
		return do.MustInvoke[param.Fetcher](i).FetchAll(ctx, os.Getenv("PROMPTS_PARAM"))
	})
	do.ProvideNamedValue[string](injector, "bucket", os.Getenv("BUCKET"))
	do.ProvideNamedValue[string](injector, "distribution", os.Getenv("DISTRIBUTION"))

	do.Provide[*Handler](injector, func(i *do.Injector) (*Handler, error) {
		return &Handler{
			Randomizer:  do.MustInvoke[*prompt.Randomizer](i),
			Generator:   do.MustInvoke[image.Generator](i),
			Templator:   do.MustInvoke[*page.Templator](i),
			Uploader:    do.MustInvoke[store.Uploader](i),
			Invalidator: do.MustInvoke[store.Invalidator](i),
		}, nil
	})

	if _, ok := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); ok {
		handler := do.MustInvoke[*Handler](injector)
		lambda.StartWithOptions(handler.HandleRequest, lambda.WithContext(ctx), lambda.WithEnableSIGTERM(func() {
			_ = injector.Shutdown()
		}))
	} else {
		do.OverrideValue[store.Uploader](injector, &store.FileUploader{})
		do.OverrideValue[store.Invalidator](injector, &store.NoopInvalidator{})

		var params Params
		if len(os.Args) > 1 {
			if err := json.Unmarshal([]byte(os.Args[1]), &params); err != nil {
				panic(err)
			}
		}

		handler := do.MustInvoke[*Handler](injector)
		params, err := handler.HandleRequest(ctx, params)
		if err != nil {
			panic(err)
		}

		if err := json.NewEncoder(os.Stdout).Encode(params); err != nil {
			panic(err)
		}

		if err := injector.Shutdown(); err != nil {
			panic(err)
		}
	}
}
