package param

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/go-logr/logr"
	"github.com/samber/lo"
)

type ParameterStoreFetcher struct {
	Client *ssm.Client
}

func (f *ParameterStoreFetcher) Fetch(ctx context.Context, path string) (string, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("path", path)
	log.Info("fetching single parameter")

	out, err := f.Client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Parameter.Value), nil
}

func (f *ParameterStoreFetcher) FetchAll(ctx context.Context, path string) ([]string, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("path", path)
	log.Info("fetching all parameters")

	out, err := f.Client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
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
