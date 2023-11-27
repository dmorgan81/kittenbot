package post

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type RedditPoster struct {
	client    *reddit.Client
	subreddit string
}

func NewRedditPoster(i *do.Injector) (Poster, error) {
	creds := reddit.Credentials{
		ID:       do.MustInvokeNamed[string](i, "reddit_client_id"),
		Secret:   do.MustInvokeNamed[string](i, "reddit_client_secret"),
		Username: do.MustInvokeNamed[string](i, "reddit_username"),
		Password: do.MustInvokeNamed[string](i, "reddit_password"),
	}
	client, err := reddit.NewClient(creds, reddit.WithHTTPClient(do.MustInvoke[*http.Client](i)))
	if err != nil {
		return nil, err
	}

	subreddit := do.MustInvokeNamed[string](i, "subreddit")

	return &RedditPoster{
		client:    client,
		subreddit: subreddit,
	}, nil
}

func (p *RedditPoster) Post(ctx context.Context, params Params) error {
	logger := log.FromContextOrDiscard(ctx)
	logger.Info("posting to reddit", "subreddit", p.subreddit, "params", params)

	_, _, err := p.client.Post.SubmitLink(ctx, reddit.SubmitLinkRequest{
		Subreddit:   p.subreddit,
		Title:       fmt.Sprintf("%s - %s:%s:%s", params.Date, params.Prompt, params.Model, params.Seed),
		URL:         fmt.Sprintf("https://kittenbot.io/%s.png", params.Date),
		SendReplies: lo.ToPtr(false),
	})
	return err
}
