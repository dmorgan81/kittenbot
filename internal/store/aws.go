package store

import (
	"bytes"
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-logr/logr"
)

type S3Uploader struct {
	Client *s3.Client
	Bucket string
}

func (u *S3Uploader) Upload(ctx context.Context, params UploadParams) error {
	log := logr.FromContextOrDiscard(ctx).WithValues(
		"name", params.Name,
		"content-type", params.ContentType,
		"metadata", params.Metadata,
		"bucket", u.Bucket,
	)
	log.Info("uploading to s3")

	_, err := u.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(u.Bucket),
		Key:          aws.String(params.Name),
		ContentType:  aws.String(params.ContentType),
		Body:         bytes.NewReader(params.Data),
		Metadata:     params.Metadata,
		StorageClass: s3types.StorageClassIntelligentTiering,
	})
	return err
}

type CloudFrontInvalidator struct {
	Client       *cloudfront.Client
	Distribution string
}

func (i *CloudFrontInvalidator) Invalidate(ctx context.Context, paths []string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("paths", paths, "distribution", i.Distribution)
	log.Info("invalidating paths in cloudfront")

	_, err := i.Client.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(i.Distribution),
		InvalidationBatch: &cftypes.InvalidationBatch{
			CallerReference: aws.String(time.Now().UTC().Format("20060102150405")),
			Paths: &cftypes.Paths{
				Quantity: aws.Int32(int32(len(paths))),
				Items:    paths,
			},
		},
	})
	return err
}
