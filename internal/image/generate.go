package image

import "context"

type Params struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Seed   string `json:"seed,omitempty"`
}

type Generator interface {
	Generate(context.Context, Params) ([]byte, string, error)
}
