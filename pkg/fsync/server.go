package fsync

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/pjoc-team/fsync/internal/viper"
	"github.com/pjoc-team/fsync/pkg/storage/api"
	"github.com/pjoc-team/threadpool"
	"github.com/pjoc-team/tracing/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	confFile = "fsync.yaml"
)

var (
	// ErrInvalidPath invalid params
	ErrInvalidPath = errors.New("invalid confPath")
)

// Config config data
type Config struct {
	FileInitialized bool `yaml:"fileInitialized"`
}

type server struct {
	rootPath   string
	options    *foptions
	storage    api.FileStorage
	watcher    *fsnotify.Watcher
	ctx        context.Context
	cancelFunc context.CancelFunc
	conf       *Config
}

// NewServer create server
func NewServer(
	ctx context.Context, rootPath string, storage api.FileStorage, opts ...Option,
) (_ SyncServer, err error) {
	ctx, cancel := context.WithCancel(ctx)
	log := logger.ContextLog(ctx)

	watcher, err2 := fsnotify.NewWatcher()
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	if err2 != nil {
		log.Errorf("failed create watcher error: %v", err2.Error())
		cancel()
		return nil, err2
	}

	o, err2 := newFoptions(opts...)
	if err2 != nil {
		err2 := watcher.Close()
		if err2 != nil {
			log.Errorf("failed to close watcher, error: %v", err2.Error())
		}
		cancel()
		return nil, err2
	}

	if o.ConfPath == "" {
		return nil, ErrInvalidPath
	}

	fp := filepath.Join(o.ConfPath, confFile)
	cs, err := viper.NewConf(fp)
	if err != nil {
		log.Errorf("failed to init conf: %v, error: %v", fp, err.Error())
		return nil, err
	}
	conf := &Config{}
	err = cs.UnmarshalConfig(conf)
	if err != nil {
		log.Errorf("failed to init conf: %v, error: %v", fp, err.Error())
		return nil, err
	}

	svr := &server{
		watcher:    watcher,
		rootPath:   rootPath,
		options:    &o,
		storage:    storage,
		ctx:        ctx,
		cancelFunc: cancel,
		conf:       conf,
	}
	err2 = svr.AddPath(rootPath)
	if err2 != nil {
		log.Errorf("failed create watcher of file: %v, error: %v", rootPath, err2.Error())
	}
	go svr.watchFile()
	return svr, nil
}

func (s *server) AddPath(path string) error {
	log := logger.ContextLog(s.ctx)
	_, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			log.Errorf("failed mkdir: %v, error: %v", path, err.Error())
			return err
		}
	}
	err = s.watcher.Add(path)
	if err != nil {
		log.Errorf("failed create watcher of file: %v, error: %v", path, err.Error())
		return err
	}
	pool, err := threadpool.NewPool(s.ctx, s.options.ThreadPoolSize)
	if err != nil {
		log.Errorf("failed to create ThreadPool, error: %v", err.Error())
	}

	wg := sync.WaitGroup{}

	err = filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err := s.watcher.Add(subPath)
			if err != nil {
				log.Errorf("failed to watch file: %v", subPath)
			}
			return nil
		}
		if !s.options.InitUpload {
			return nil
		}
		if pool != nil {
			pool.Run(func() {
				wg.Add(1)
				defer wg.Done()
				err = s.uploadFile(subPath)
				if err != nil {
					log.Errorf("failed to upload file: %v", subPath)
				}
			})
		} else {
			err = s.uploadFile(subPath)
			if err != nil {
				log.Errorf("failed to upload file: %v", subPath)
			}
		}
		return nil
	})
	wg.Wait()
	if err != nil {
		log.Errorf("failed create watcher of file: %v, error: %v", path, err.Error())
		return err
	}

	s.conf.FileInitialized = true

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
	log.Println("upload file:", file)
	reader, err := os.Open(file)
	if err != nil {
		log.Errorf("failed to open file: %v error: %v", file, err.Error())
		return err
	}
	stat, err := reader.Stat()
	if err != nil {
		log.Errorf("failed to open file: %v error: %v", file, err.Error())
		return err
	}
	if stat.IsDir() {
		log.Debugf("file: %v is dir, skip upload", file)
		return nil
	}

	absolutePath := strings.TrimPrefix(file, s.rootPath)
	path := filepath.Join("/", absolutePath)
	writer, err := s.storage.Create(s.ctx, path)
	if err != nil {
		log.Errorf("failed to create file: %v error: %v", file, err.Error())
		return err
	}
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
