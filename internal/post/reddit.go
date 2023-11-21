package post

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/sethjones/go-reddit/v2/reddit"
)

type RedditPoster struct {
	client    *reddit.Client
	subreddit string
}

func NewRedditPoster(i *do.Injector) (Poster, error) {
	id := do.MustInvokeNamed[string](i, "reddit_client_id")
	secret := do.MustInvokeNamed[string](i, "reddit_client_secret")
	subreddit := do.MustInvokeNamed[string](i, "subreddit")

	info, _ := debug.ReadBuildInfo()
	setting := lo.FindOrElse(info.Settings, debug.BuildSetting{Value: "unknown"}, func(s debug.BuildSetting) bool {
		return s.Key == "vcs.revision"
	})

	client, err := reddit.NewClient(reddit.Credentials{ID: id, Secret: secret},
		reddit.WithApplicationOnlyOAuth(true), reddit.WithUserAgent("web:kittenbot:"+setting.Value))
	if err != nil {
		return nil, err
	}

	return &RedditPoster{client, subreddit}, nil
}

func (p *RedditPoster) Post(ctx context.Context, params Params) error {
	log.FromContextOrDiscard(ctx).Info("posting to reddit", "subreddit", p.subreddit)
	_, _, err := p.client.Post.SubmitLink(ctx, reddit.SubmitLinkRequest{
		Subreddit:   p.subreddit,
		Title:       fmt.Sprintf("%s - %s:%s:%s", params.Date, params.Prompt, params.Model, params.Seed),
		URL:         fmt.Sprintf("https://kittenbot.io/%s.html", params.Date),
		SendReplies: lo.ToPtr(false),
	})
	return err
}
