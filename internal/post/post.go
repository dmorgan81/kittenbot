package post

import "context"

type Params struct {
	Date   string
	Model  string
	Prompt string
	Seed   string
}

type Poster interface {
	Post(context.Context, Params) error
}
