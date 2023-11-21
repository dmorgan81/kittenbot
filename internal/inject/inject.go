package inject

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/dmorgan81/kittenbot/internal/feed"
	"github.com/dmorgan81/kittenbot/internal/handler"
	"github.com/dmorgan81/kittenbot/internal/image"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/dmorgan81/kittenbot/internal/page"
	"github.com/dmorgan81/kittenbot/internal/param"
	"github.com/dmorgan81/kittenbot/internal/prompt"
	"github.com/dmorgan81/kittenbot/internal/store"
	"github.com/samber/do"
)

func Setup(ctx context.Context) *do.Injector {
	log := log.FromContextOrDiscard(ctx)

	injector := do.NewWithOpts(&do.InjectorOpts{
		Logf: func(format string, args ...any) {
			log.Info(fmt.Sprintf(format, args))
		},
	})
	do.Provide[aws.Config](injector, func(i *do.Injector) (aws.Config, error) {
		return config.LoadDefaultConfig(ctx)
	})
	do.Provide[*ssm.Client](injector, func(i *do.Injector) (*ssm.Client, error) {
		return ssm.NewFromConfig(do.MustInvoke[aws.Config](i)), nil
	})
	do.Provide[*s3.Client](injector, func(i *do.Injector) (*s3.Client, error) {
		return s3.NewFromConfig(do.MustInvoke[aws.Config](i)), nil
	})
	do.Provide[*cloudfront.Client](injector, func(i *do.Injector) (*cloudfront.Client, error) {
		return cloudfront.NewFromConfig(do.MustInvoke[aws.Config](i)), nil
	})
	do.ProvideValue[*http.Client](injector, http.DefaultClient)

	do.Provide[param.Fetcher](injector, param.NewParameterStoreFetcher)
	do.Provide[*prompt.Randomizer](injector, prompt.NewRandomizer)
	do.Provide[image.Generator](injector, image.NewDezgoGenerator)
	do.Provide[store.Uploader](injector, store.NewS3Uploader)
	do.Provide[store.Invalidator](injector, store.NewCloudFrontInvalidator)
	do.Provide[*page.Templator](injector, page.NewTemplator)
	do.Provide[*feed.Generator](injector, feed.NewS3Generator)

	do.ProvideNamed[string](injector, "dezgo_key", func(i *do.Injector) (string, error) {
		return do.MustInvoke[param.Fetcher](i).Fetch(ctx, os.Getenv("DEZGO_KEY_PARAM"))
	})
	do.ProvideNamed[[]string](injector, "prompts", func(i *do.Injector) ([]string, error) {
		return do.MustInvoke[param.Fetcher](i).FetchAll(ctx, os.Getenv("PROMPTS_PARAM"))
	})
	do.ProvideNamed[string](injector, "reddit_client_id", func(i *do.Injector) (string, error) {
		return do.MustInvoke[param.Fetcher](i).Fetch(ctx, os.Getenv("REDDIT_CLIENT_ID_PARAM"))
	})
	do.ProvideNamed[string](injector, "reddit_client_secret", func(i *do.Injector) (string, error) {
		return do.MustInvoke[param.Fetcher](i).Fetch(ctx, os.Getenv("REDDIT_CLIENT_SECRET_PARAM"))
	})
	do.ProvideNamedValue[string](injector, "bucket", os.Getenv("BUCKET"))
	do.ProvideNamedValue[string](injector, "distribution", os.Getenv("DISTRIBUTION"))
	do.ProvideNamedValue[string](injector, "subreddit", os.Getenv("SUBREDDIT"))

	do.Provide[*handler.Handler](injector, handler.NewHandler)

	return injector
}
