package reddit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"

	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/dmorgan81/kittenbot/internal/post"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type creds struct {
	id, secret string
}

type Poster struct {
	client    *http.Client
	creds     creds
	userAgent string
	subreddit string
}

func NewPoster(i *do.Injector) (post.Poster, error) {
	id := do.MustInvokeNamed[string](i, "reddit_client_id")
	secret := do.MustInvokeNamed[string](i, "reddit_client_secret")
	subreddit := do.MustInvokeNamed[string](i, "subreddit")
	username := do.MustInvokeNamed[string](i, "reddit_username")
	client := do.MustInvoke[*http.Client](i)

	info, _ := debug.ReadBuildInfo()
	setting := lo.FindOrElse(info.Settings, debug.BuildSetting{Value: "unknown"}, func(s debug.BuildSetting) bool {
		return s.Key == "vcs.revision"
	})
	userAgent := fmt.Sprintf("web:kittenbot:%s (by /u/%s", setting.Value, username)

	return &Poster{
		client:    client,
		creds:     creds{id, secret},
		userAgent: userAgent,
		subreddit: subreddit,
	}, nil
}

func (p *Poster) Post(ctx context.Context, params post.Params) error {
	logger := log.FromContextOrDiscard(ctx)
	logger.Info("posting to reddit", "subreddit", p.subreddit)

	token, err := p.getAccessToken(ctx)
	if err != nil {
		return err
	}
	logger.Debug("fetched access token", "token", token)

	return p.submit(ctx, params, token)
}

func (p *Poster) getAccessToken(ctx context.Context) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest(http.MethodPost, "https://www.reddit.com/api/v1/access_token",
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", p.userAgent)
	req.SetBasicAuth(p.creds.id, p.creds.secret)

	resp, err := p.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body["access_token"].(string), nil
}

func (p *Poster) submit(ctx context.Context, params post.Params, token string) error {
	data := url.Values{}
	data.Set("api_type", "json") // https://www.reddit.com/dev/api/oauth#POST_api_submit
	data.Set("kind", "link")
	data.Set("sr", p.subreddit)
	data.Set("title", fmt.Sprintf("%s - %s:%s:%s", params.Date, params.Prompt, params.Model, params.Date))
	data.Set("url", fmt.Sprintf("https://kittenbot.io/%s.html", params.Date))

	req, err := http.NewRequest(http.MethodPost, "https://oauth.reddit.com/api/submit",
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))

	resp, err := p.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
