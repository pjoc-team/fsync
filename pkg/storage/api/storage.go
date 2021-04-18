package api

import (
"context"
"io"
)

//go:generate mockgen -source ./storage.go -package mock -destination ./mock/mock.go

// FileStorage file storage api
type FileStorage interface {
	// Create create file and returns FileUpload object
	Create(ctx context.Context, path string, opts ...Option) (io.WriteCloser, error)

	// Get get file
	Get(ctx context.Context, path string, options ...Option) (io.ReadCloser, error)

	// Info get file info
	Info(ctx context.Context, path string) (*FileInfo, error)
}

// FileInfo file info
type FileInfo struct {
	Path string
	FileName string
	Size int64
}
