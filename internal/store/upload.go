package store

import (
	"context"
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
