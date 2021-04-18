package fs

import (
	"context"
	"github.com/pjoc-team/fsync/pkg/storage/api"
	"io"
	"os"
	"path/filepath"
)

type localFileStorage struct {
	rootPath string
}

// NewLocalFileStorage create local file storage
func NewLocalFileStorage(rootPath string) (api.FileStorage, error) {
	if rootPath != "" {
		err := os.MkdirAll(rootPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return &localFileStorage{rootPath: rootPath}, nil
}

func (l *localFileStorage) Create(
	ctx context.Context, path string,
	opts ...api.Option,
) (io.WriteCloser, error) {
	f := filepath.Join(l.rootPath, path)
	err := os.MkdirAll(l.rootPath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	fi, err := os.Create(f)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func (l *localFileStorage) Get(
	ctx context.Context, path string, options ...api.Option,
) (io.ReadCloser, error) {
	f := filepath.Join(l.rootPath, path)
	fi, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func (l *localFileStorage) Info(ctx context.Context, path string) (*api.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileInfo := &api.FileInfo{
		Path:     path,
		FileName: stat.Name(),
		Size:     stat.Size(),
	}

	return fileInfo, nil
}
