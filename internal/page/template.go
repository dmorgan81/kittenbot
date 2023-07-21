package page

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"

	"github.com/go-logr/logr"
	"github.com/samber/do"
)

//go:embed assets/latest.html
var latestTmpl string

type Params struct {
	Image  string
	Model  string
	Prompt string
	Seed   string

	Prev string
	Next string
}

type Templator struct {
	tmpl *template.Template
}

func NewTemplator(i *do.Injector) (*Templator, error) {
	tmpl := template.Must(template.New("latest").Parse(latestTmpl))
	return &Templator{tmpl}, nil
}

func (g *Templator) Template(ctx context.Context, params Params) ([]byte, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("templator")
	log.Info("generating page")

	var data bytes.Buffer
	if err := g.tmpl.Execute(&data, params); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}
