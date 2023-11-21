package feed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/gorilla/feeds"
	"github.com/samber/do"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

type Generator struct {
	client *s3.Client
	bucket string
}

func NewS3Generator(i *do.Injector) (*Generator, error) {
	client := do.MustInvoke[*s3.Client](i)
	bucket := do.MustInvokeNamed[string](i, "bucket")
	return &Generator{client, bucket}, nil
}

func (g *Generator) Generate(ctx context.Context) ([]byte, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("feed")
	log.Info("generating rss feed")

	feed := feeds.Feed{
		Title:       "KittenBot",
		Description: "Daily AI Generated Kittens",
		Link:        &feeds.Link{Href: "https://kittenbot.io"},
		Updated:     time.Now(),
	}

	pager := s3.NewListObjectsV2Paginator(g.client, &s3.ListObjectsV2Input{
		Bucket: &g.bucket,
	})

	items := make(chan *feeds.Item)
	defer close(items)

	go func(items <-chan *feeds.Item) {
		for i := range items {
			feed.Add(i)
		}
	}(items)

	group, ctx := errgroup.WithContext(ctx)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		objs := lo.Filter(page.Contents, func(o s3types.Object, _ int) bool {
			return strings.HasSuffix(*o.Key, ".png") && !strings.HasPrefix(*o.Key, "latest")
		})

		for _, obj := range objs {
			obj := obj
			group.Go(func() error {
				out, err := g.client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &g.bucket,
					Key:    obj.Key,
				})
				if err != nil {
					return err
				}

				meta := out.Metadata
				items <- &feeds.Item{
					Title:   fmt.Sprintf("%s:%s:%s", meta["prompt"], meta["model"], meta["seed"]),
					Link:    &feeds.Link{Href: fmt.Sprintf("https://kittenbot.io/%s.png", meta["date"])},
					Updated: *out.LastModified,
				}
				return nil
			})
		}
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	feed.Sort(func(a, b *feeds.Item) bool {
		return a.Updated.Before(b.Updated)
	})
	rss, err := feed.ToRss()
	return []byte(rss), err
}
