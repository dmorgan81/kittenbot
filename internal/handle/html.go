package handle

import (
	"bytes"
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dmorgan81/kittenbot/internal/log"
	"github.com/dmorgan81/kittenbot/internal/page"
	"github.com/samber/do"
)

var urlRegexp = regexp.MustCompile(`^https://.+\.amazonaws\.com/(?P<key>.+?)\.html(?:\?.*)?$`)

type objectContext struct {
	Url   string `json:"inputS3Url"`
	Route string `json:"outputRoute"`
	Token string `json:"outputToken"`
}

type HtmlRequest struct {
	Id         string        `json:"xAmzRequestId"`
	GetContext objectContext `json:"getObjectContext"`
}

type HtmlHandler struct {
	client    *s3.Client
	bucket    string
	templator *page.Templator
}

func NewHtmlHandler(i *do.Injector) (*HtmlHandler, error) {
	return &HtmlHandler{
		client:    do.MustInvoke[*s3.Client](i),
		bucket:    do.MustInvokeNamed[string](i, "bucket"),
		templator: do.MustInvoke[*page.Templator](i),
	}, nil
}

func (h *HtmlHandler) Handle(ctx context.Context, request HtmlRequest) error {
	log := log.FromContextOrDiscard(ctx).WithGroup("HtmlHandler").With("request", request)
	matches := urlRegexp.FindStringSubmatch(request.GetContext.Url)
	key := matches[urlRegexp.SubexpIndex("key")]
	log.Info("handling lambda request", "key", key)

	out, err := h.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(h.bucket),
		Key:    aws.String(key + ".png"),
	})
	if err != nil {
		return err
	}

	html, err := h.templator.Template(ctx, page.Params{
		Image:  out.Metadata["date"] + ".png",
		Model:  out.Metadata["model"],
		Prompt: out.Metadata["prompt"],
		Seed:   out.Metadata["seed"],
	})
	if err != nil {
		return err
	}

	_, err = h.client.WriteGetObjectResponse(ctx, &s3.WriteGetObjectResponseInput{
		RequestRoute: aws.String(request.GetContext.Route),
		RequestToken: aws.String(request.GetContext.Token),

		Body:          bytes.NewReader(html),
		ContentLength: int64(len(html)),
		ContentType:   aws.String("text/html"),
		ETag:          out.ETag,
		Expires:       out.Expires,
		LastModified:  out.LastModified,
		Metadata:      out.Metadata,
		StatusCode:    200,
	})
	return err
}
