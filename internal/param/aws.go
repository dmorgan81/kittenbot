package param

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type ParameterStoreFetcher struct {
	client *ssm.Client
}

func NewParameterStoreFetcher(i *do.Injector) (Fetcher, error) {
	return &ParameterStoreFetcher{client: do.MustInvoke[*ssm.Client](i)}, nil
}

func (f *ParameterStoreFetcher) Fetch(ctx context.Context, path string) (string, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("parameter store").With("path", path)
	log.Info("fetching single parameter")

	out, err := f.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Parameter.Value), nil
}

func (f *ParameterStoreFetcher) FetchAll(ctx context.Context, path string) ([]string, error) {
	log := log.FromContextOrDiscard(ctx).WithGroup("parameter store").With("path", path)
	log.Info("fetching all parameters")

	out, err := f.client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return lo.Map(out.Parameters, func(p types.Parameter, _ int) string {
		return aws.ToString(p.Value)
	}), nil
}
