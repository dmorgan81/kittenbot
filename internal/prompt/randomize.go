package prompt

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/dmorgan81/kittenbot/internal/param"
	"github.com/go-logr/logr"
)

type Randomizer struct {
	Fetcher param.Fetcher
	Path    string

	rnd  *rand.Rand
	once sync.Once
}

func (r *Randomizer) Randomize(ctx context.Context) (string, string, error) {
	r.once.Do(func() {
		r.rnd = rand.New(rand.NewSource(time.Now().UTC().Unix()))
	})

	log := logr.FromContextOrDiscard(ctx).WithName("randomizer")
	log.Info("getting random model and prompt")

	prompts, err := r.Fetcher.FetchAll(ctx, r.Path)
	if err != nil {
		return "", "", err
	}

	idx := r.rnd.Intn(len(prompts))
	pair := strings.Split(prompts[idx], "|")
	return pair[0], pair[1], nil
}
