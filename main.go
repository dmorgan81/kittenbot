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
	ctx = logr.NewContext(ctx, stdr.New(nil))
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
	ctx := logr.NewContext(context.Background(), stdr.New(log.New(os.Stderr, "", 0)))

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}

	fetcher := &param.ParameterStoreFetcher{
		Client: ssm.NewFromConfig(cfg),
	}

	randomizer := &prompt.Randomizer{
		Fetcher: fetcher,
		Path:    os.Getenv("PROMPTS_PARAM"),
	}

	var generator image.Generator
	{
		key, err := fetcher.Fetch(ctx, os.Getenv("DEZGO_KEY_PARAM"))
		if err != nil {
			panic(err)
		}

		generator = &image.DezgoGenerator{
			Client: http.DefaultClient,
			Key:    key,
		}
	}

	uploader := &store.S3Uploader{
		Client: s3.NewFromConfig(cfg),
		Bucket: os.Getenv("BUCKET"),
	}

	invalidator := &store.CloudFrontInvalidator{
		Client:       cloudfront.NewFromConfig(cfg),
		Distribution: os.Getenv("DISTRIBUTION"),
	}

	handler := &Handler{
		Randomizer:  randomizer,
		Generator:   generator,
		Templator:   &page.Templator{},
		Uploader:    uploader,
		Invalidator: invalidator,
	}

	if _, ok := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); ok {
		lambda.StartWithOptions(handler.HandleRequest, lambda.WithContext(ctx))
	} else {
		var params Params
		if len(os.Args) > 1 {
			if err := json.Unmarshal([]byte(os.Args[1]), &params); err != nil {
				panic(err)
			}
		}
		params, err := handler.HandleRequest(ctx, params)
		if err != nil {
			panic(err)
		}
		fmt.Println(params)
	}
}
