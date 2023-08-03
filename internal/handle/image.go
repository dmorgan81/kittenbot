package handle

import (
	"context"
	"time"

	"github.com/dmorgan81/kittenbot/internal/image"
	"github.com/dmorgan81/kittenbot/internal/page"
	"github.com/dmorgan81/kittenbot/internal/prompt"
	"github.com/dmorgan81/kittenbot/internal/store"
	"github.com/go-logr/logr"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type ImageInput struct {
	Date   string `json:"date,omitempty"`
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Seed   string `json:"seed,omitempty"`
}

func (i ImageInput) toImageParams() image.Params {
	return image.Params{
		Model:  i.Model,
		Prompt: i.Prompt,
		Seed:   i.Seed,
	}
}

func (i ImageInput) toPageParams() page.Params {
	return page.Params{
		Image:  i.Date + ".png",
		Model:  i.Model,
		Prompt: i.Prompt,
		Seed:   i.Seed,
	}
}

func (i ImageInput) toMetadata() map[string]string {
	return map[string]string{
		"Date":   i.Date,
		"Model":  i.Model,
		"Prompt": i.Prompt,
		"Seed":   i.Seed,
	}
}

type ImageOutput ImageInput

type ImageHandler struct {
	randomizer  *prompt.Randomizer
	generator   image.Generator
	templator   *page.Templator
	uploader    store.Uploader
	invalidator store.Invalidator
}

func NewImageHandler(i *do.Injector) (*ImageHandler, error) {
	return &ImageHandler{
		randomizer:  do.MustInvoke[*prompt.Randomizer](i),
		generator:   do.MustInvoke[image.Generator](i),
		templator:   do.MustInvoke[*page.Templator](i),
		uploader:    do.MustInvoke[store.Uploader](i),
		invalidator: do.MustInvoke[store.Invalidator](i),
	}, nil
}

func (h *ImageHandler) Handle(ctx context.Context, input ImageInput) (ImageOutput, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("handler").WithValues("input", input)
	log.Info("handling lambda invocation")

	if input.Model == "" || input.Prompt == "" {
		model, prompt, err := h.randomizer.Randomize(ctx)
		if err != nil {
			return ImageOutput{}, err
		}
		input.Model = lo.Ternary(input.Model != "", input.Model, model)
		input.Prompt = lo.Ternary(input.Prompt != "", input.Prompt, prompt)
	}

	if input.Date == "" {
		input.Date = time.Now().UTC().Format("20060102")
	}

	img, seed, err := h.generator.Generate(ctx, input.toImageParams())
	if err != nil {
		return ImageOutput{}, err
	}
	input.Seed = seed

	html, err := h.templator.Template(ctx, input.toPageParams())
	if err != nil {
		return ImageOutput{}, err
	}

	metadata := input.toMetadata()
	uploads := []store.UploadParams{
		{Name: input.Date + ".png", Data: img, ContentType: "image/png", Metadata: metadata},
		{Name: "latest.png", Data: img, ContentType: "image/png", Metadata: metadata},
		{Name: input.Date + ".html", Data: html, ContentType: "text/html", Metadata: metadata},
		{Name: "latest.html", Data: html, ContentType: "text/html", Metadata: metadata},
	}
	for _, u := range uploads {
		if err := h.uploader.Upload(ctx, u); err != nil {
			return ImageOutput{}, err
		}
	}

	paths := lo.Map(uploads, func(u store.UploadParams, _ int) string { return "/" + u.Name })
	if err := h.invalidator.Invalidate(ctx, paths); err != nil {
		return ImageOutput{}, err
	}

	return ImageOutput(input), nil
}
