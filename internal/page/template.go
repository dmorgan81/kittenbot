package page

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"sync"

	"github.com/go-logr/logr"
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
	once sync.Once
}

func (g *Templator) Template(ctx context.Context, params Params) ([]byte, error) {
	g.once.Do(func() {
		g.tmpl = template.Must(template.New("latest").Parse(latestTmpl))
	})

	log := logr.FromContextOrDiscard(ctx).WithName("templator")
	log.Info("generating page")

	var data bytes.Buffer
	if err := g.tmpl.Execute(&data, params); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}
