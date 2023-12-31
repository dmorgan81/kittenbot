package page

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"

	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
)

//go:embed assets/latest.html
var latestTmpl string

type Params struct {
	Image  string
	Model  string
	Prompt string
	Seed   string
}

type Templator struct {
	tmpl *template.Template
}

func NewTemplator(i *do.Injector) (*Templator, error) {
	tmpl := template.Must(template.New("latest").Parse(latestTmpl))
	return &Templator{tmpl}, nil
}

func (g *Templator) Template(ctx context.Context, params Params) ([]byte, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("templator").With("params", params)
	log.Info("generating page")

	var data bytes.Buffer
	if err := g.tmpl.Execute(&data, params); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}
