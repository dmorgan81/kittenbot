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
	"github.com/samber/do"
)

type S3Uploader struct {
	client *s3.Client
	bucket string
}

func NewS3Uploader(i *do.Injector) (Uploader, error) {
	client := do.MustInvoke[*s3.Client](i)
	bucket := do.MustInvokeNamed[string](i, "bucket")
	return &S3Uploader{client, bucket}, nil
}

func (u *S3Uploader) Upload(ctx context.Context, params UploadParams) error {
	log := logr.FromContextOrDiscard(ctx).WithName("s3 uploader").WithValues(
		"name", params.Name,
		"content-type", params.ContentType,
		"metadata", params.Metadata,
		"bucket", u.bucket,
	)
	log.Info("uploading")

	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(u.bucket),
		Key:          aws.String(params.Name),
		ContentType:  aws.String(params.ContentType),
		Body:         bytes.NewReader(params.Data),
		Metadata:     params.Metadata,
		StorageClass: s3types.StorageClassIntelligentTiering,
	})
	return err
}

type CloudFrontInvalidator struct {
	client       *cloudfront.Client
	distribution string
}

func NewCloudFrontInvalidator(i *do.Injector) (Invalidator, error) {
	client := do.MustInvoke[*cloudfront.Client](i)
	distribution := do.MustInvokeNamed[string](i, "distribution")
	return &CloudFrontInvalidator{client, distribution}, nil
}

func (i *CloudFrontInvalidator) Invalidate(ctx context.Context, paths []string) error {
	log := logr.FromContextOrDiscard(ctx).WithName("cloudfront invalidator").WithValues(
		"paths", paths,
		"distribution", i.distribution,
	)
	log.Info("invalidating paths in cloudfront")

	_, err := i.client.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(i.distribution),
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
