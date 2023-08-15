package image

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
)

type DezgoGenerator struct {
	client *http.Client
	key    string
}

func NewDezgoGenerator(i *do.Injector) (Generator, error) {
	client := do.MustInvoke[*http.Client](i)
	key := do.MustInvokeNamed[string](i, "dezgo_key")
	return &DezgoGenerator{client, key}, nil
}

func (g *DezgoGenerator) Generate(ctx context.Context, params Params) ([]byte, string, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("dezgo").With("params", params)
	log.Info("generating image via api.dezgo.com")

	body, err := json.Marshal(params)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.dezgo.com/text2image", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Dezgo-Key", g.key)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	seed := resp.Header.Get("x-input-seed")
	log.Info("received image via api.dezgo.com", "seed", seed)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return data, seed, nil
}
