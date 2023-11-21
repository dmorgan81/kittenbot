package handler

import (
	"context"
	"time"

	"github.com/dmorgan81/kittenbot/internal/image"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/dmorgan81/kittenbot/internal/page"
	"github.com/dmorgan81/kittenbot/internal/prompt"
	"github.com/dmorgan81/kittenbot/internal/store"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type Input struct {
	Date   string `json:"date,omitempty"`
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Seed   string `json:"seed,omitempty"`
}

func (i Input) toImageParams() image.Params {
	return image.Params{
		Model:  i.Model,
		Prompt: i.Prompt,
		Seed:   i.Seed,
	}
}

func (i Input) toPageParams() page.Params {
	return page.Params{
		Image:  i.Date + ".png",
		Model:  i.Model,
		Prompt: i.Prompt,
		Seed:   i.Seed,
	}
}

func (i Input) toMetadata() map[string]string {
	return map[string]string{
		"date":   i.Date,
		"model":  i.Model,
		"prompt": i.Prompt,
		"seed":   i.Seed,
	}
}

type Output Input

type Handler struct {
	randomizer  *prompt.Randomizer
	generator   image.Generator
	uploader    store.Uploader
	invalidator store.Invalidator
	templator   *page.Templator
}

func NewHandler(i *do.Injector) (*Handler, error) {
	return &Handler{
		randomizer:  do.MustInvoke[*prompt.Randomizer](i),
		generator:   do.MustInvoke[image.Generator](i),
		uploader:    do.MustInvoke[store.Uploader](i),
		invalidator: do.MustInvoke[store.Invalidator](i),
		templator:   do.MustInvoke[*page.Templator](i),
	}, nil
}

func (h *Handler) Handle(ctx context.Context, input Input) (Output, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("Handler").With("input", input)
	log.Info("handling lambda invocation")

	if input.Model == "" || input.Prompt == "" {
		model, prompt, err := h.randomizer.Randomize(ctx)
		if err != nil {
			return Output{}, err
		}
		input.Model = lo.Ternary(input.Model != "", input.Model, model)
		input.Prompt = lo.Ternary(input.Prompt != "", input.Prompt, prompt)
	}

	latest := false
	if input.Date == "" {
		input.Date = time.Now().UTC().Format("20060102")
		latest = true
	}

	img, seed, err := h.generator.Generate(ctx, input.toImageParams())
	if err != nil {
		return Output{}, err
	}
	input.Seed = seed

	html, err := h.templator.Template(ctx, input.toPageParams())
	if err != nil {
		return Output{}, err
	}

	metadata := input.toMetadata()
	uploads := []store.UploadParams{
		{
			Name:        input.Date + ".png",
			Data:        img,
			ContentType: "image/png",
			Metadata:    metadata,
		},
		{
			Name:        input.Date + ".html",
			Data:        html,
			ContentType: "text/html",
			Metadata:    metadata,
		},
	}
	if latest {
		uploads = append(uploads,
			store.UploadParams{
				Name:        "latest.png",
				Data:        img,
				ContentType: "image/png",
				Metadata:    metadata,
			},
			store.UploadParams{
				Name:        "latest.html",
				Data:        html,
				ContentType: "text/html",
				Metadata:    metadata,
			},
		)
	}
	for _, u := range uploads {
		if err := h.uploader.Upload(ctx, u); err != nil {
			return Output{}, err
		}
	}

	paths := []string{"/" + input.Date + ".png", "/" + input.Date + ".html"}
	if latest {
		paths = append(paths, "/latest.png", "/latest.html")
	}
	if err := h.invalidator.Invalidate(ctx, paths); err != nil {
		return Output{}, err
	}

	return Output(input), nil
}
