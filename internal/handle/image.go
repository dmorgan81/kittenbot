package handle

import (
	"context"
	"time"

	"github.com/dmorgan81/kittenbot/internal/image"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/dmorgan81/kittenbot/internal/prompt"
	"github.com/dmorgan81/kittenbot/internal/store"
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

func (i ImageInput) toMetadata() map[string]string {
	return map[string]string{
		"date":   i.Date,
		"model":  i.Model,
		"prompt": i.Prompt,
		"seed":   i.Seed,
	}
}

type ImageOutput ImageInput

type ImageHandler struct {
	randomizer  *prompt.Randomizer
	generator   image.Generator
	uploader    store.Uploader
	invalidator store.Invalidator
}

func NewImageHandler(i *do.Injector) (*ImageHandler, error) {
	return &ImageHandler{
		randomizer:  do.MustInvoke[*prompt.Randomizer](i),
		generator:   do.MustInvoke[image.Generator](i),
		uploader:    do.MustInvoke[store.Uploader](i),
		invalidator: do.MustInvoke[store.Invalidator](i),
	}, nil
}

func (h *ImageHandler) Handle(ctx context.Context, input ImageInput) (ImageOutput, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("ImageHandler").With("input", input)
	log.Info("handling lambda invocation")

	if input.Model == "" || input.Prompt == "" {
		model, prompt, err := h.randomizer.Randomize(ctx)
		if err != nil {
			return ImageOutput{}, err
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
		return ImageOutput{}, err
	}
	input.Seed = seed

	metadata := input.toMetadata()
	uploads := []store.UploadParams{
		{
			Name:        input.Date + ".png",
			Data:        img,
			ContentType: "image/png",
			Metadata:    metadata,
		},
	}
	if latest {
		uploads = append(uploads, store.UploadParams{
			Name:        "latest.png",
			Data:        img,
			ContentType: "image/png",
			Metadata:    metadata,
		})
	}
	for _, u := range uploads {
		if err := h.uploader.Upload(ctx, u); err != nil {
			return ImageOutput{}, err
		}
	}

	paths := []string{"/" + input.Date + ".png", "/" + input.Date + ".html"}
	if latest {
		paths = append(paths, "/latest.png", "/latest.html")
	}
	if err := h.invalidator.Invalidate(ctx, paths); err != nil {
		return ImageOutput{}, err
	}

	return ImageOutput(input), nil
}
