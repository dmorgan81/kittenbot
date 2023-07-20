package prompt

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/samber/do"
)

type Randomizer struct {
	prompts []string
	rnd     *rand.Rand
}

func NewRandomizer(i *do.Injector) (*Randomizer, error) {
	prompts := do.MustInvokeNamed[[]string](i, "prompts")
	rnd := rand.New(rand.NewSource(time.Now().UTC().Unix()))
	return &Randomizer{prompts, rnd}, nil
}

func (r *Randomizer) Randomize(ctx context.Context) (string, string, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("randomizer")
	log.Info("getting random model and prompt")
	idx := r.rnd.Intn(len(r.prompts))
	pair := strings.Split(r.prompts[idx], "|")
	return pair[0], pair[1], nil
}
