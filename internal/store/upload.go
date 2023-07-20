package store

import (
	"context"
	"os"

	"github.com/go-logr/logr"
)

type UploadParams struct {
	Name        string
	Data        []byte
	ContentType string
	Metadata    map[string]string
}

type Uploader interface {
	Upload(context.Context, UploadParams) error
}

type FileUploader struct{}

func (*FileUploader) Upload(ctx context.Context, params UploadParams) error {
	log := logr.FromContextOrDiscard(ctx).WithName("file")
	log.Info("writing", "file", params.Name)
	return os.WriteFile(params.Name, params.Data, 0600)
}
