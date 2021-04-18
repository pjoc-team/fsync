package fsync

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/pjoc-team/fsync/pkg/storage/api"
	"github.com/pjoc-team/tracing/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type server struct {
	rootPath   string
	options    *foptions
	storage    api.FileStorage
	watcher    *fsnotify.Watcher
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewServer create server
func NewServer(
	ctx context.Context, rootPath string, storage api.FileStorage, opts ...Option,
) (SyncServer, error) {
	ctx, cancel := context.WithCancel(ctx)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log().Errorf("failed create watcher error: %v", err.Error())
		cancel()
		return nil, err
	}

	o, err := newFoptions(opts...)
	if err != nil {
		err2 := watcher.Close()
		if err2 != nil {
			logger.Log().Errorf("failed to close watcher, error: %v", err.Error())
		}
		cancel()
		return nil, err
	}
	s := &server{
		watcher:    watcher,
		rootPath:   rootPath,
		options:    &o,
		storage:    storage,
		ctx:        ctx,
		cancelFunc: cancel,
	}
	err = s.AddPath(rootPath)
	if err != nil {
		logger.Log().Errorf("failed create watcher of file: %v, error: %v", rootPath, err.Error())
	}
	go s.watchFile()
	return s, nil
}

func (s server) AddPath(path string) error {
	_, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			logger.Log().Errorf("failed mkdir: %v, error: %v", path, err.Error())
			return err
		}
		// return err
	}

	err = s.watcher.Add(path)
	if err != nil {
		logger.Log().Errorf("failed create watcher of file: %v, error: %v", path, err.Error())
		return err
	}

	return nil
}

// Close close context
func (s *server) Close() {
	s.cancelFunc()
}

func (s *server) watchFile() {
	log := logger.Log()
	log.Infof("starting watch file")
	for {
		select {
		case event := <-s.watcher.Events:
			log.Debugf("get event: %v", event.Name)
			file := event.Name
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Infof("upload file: %v", file)
				err := s.uploadFile(file)
				if err != nil {
					log.Errorf("failed to upload file: %v, error: %v", file, err.Error())
					continue
				}
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				log.Infof("create file: %v", file)
				err := s.uploadFile(file)
				if err != nil {
					log.Errorf("failed to upload file: %v, error: %v", file, err.Error())
					continue
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Infof("removed file: %v", file)
			}


		case err := <-s.watcher.Errors:
			log.Errorf("receive err: %v", err.Error())
		case <-s.ctx.Done():
			log.Warnf("closed watcher")
			err := s.watcher.Close()
			if err != nil {
				log.Errorf("failed to close watcher, error: %v", err.Error())
			}
			return
		}
	}
}

func (s *server) uploadFile(file string) error {
	log := logger.ContextLog(s.ctx)
	log.Println("modified file:", file)
	reader, err := os.Open(file)
	if err != nil {
		log.Errorf("failed to open file: %v error: %v", file, err.Error())
		return err
	}
	absolutePath := strings.TrimPrefix(file, s.rootPath)
	path := filepath.Join("/", absolutePath)
	writer, err := s.storage.Create(s.ctx, path)
	defer func() {
		err2 := writer.Close()
		if err2 != nil {
			log.Errorf("failed close writer, error: %v", err2.Error())
		}
		err2 = reader.Close()
		if err2 != nil {
			log.Errorf("failed close reader, error: %v", err2.Error())
		}
	}()
	for {
		buf := make([]byte, 1024*1024)
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			log.Errorf("failed to read file: %v error: %v", file, err.Error())
			return err
		}
		data := buf[:n]
		_, err2 := writer.Write(data)
		if err2 != nil {
			log.Errorf("failed to write storage, error: %v", err2.Error())
			return err2
		}
		if err == io.EOF {
			return nil
		}
	}
}
