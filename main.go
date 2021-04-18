package main

import (
	"context"
	"flag"
	"github.com/pjoc-team/fsync/pkg/fsync"
	"github.com/pjoc-team/fsync/pkg/storage/api"
	oss2 "github.com/pjoc-team/fsync/pkg/storage/backend/oss"
	"github.com/pjoc-team/tracing/logger"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	path      string
	endpoint  string
	bucket    string
	secretID  string
	secretKey string
	blockSize int
	debug     bool
	confFile  string
)

// Conf config struct
type Conf struct {
	Path      string
	Endpoint  string
	Bucket    string
	SecretID  string
	SecretKey string
	BlockSize int
	Debug     bool
}

func init() {
	flag.StringVar(&confFile, "conf", "", "conf file path")
	flag.StringVar(&path, "path", "./data/", "upload path")
	flag.StringVar(&endpoint, "endpoint", "https://cos.ap-guangzhou.myqcloud.com", "endpoint")
	flag.StringVar(&bucket, "bucket", "backup-1251070767", "bucket")
	flag.StringVar(&secretID, "secret-id", "[changeSecretID]", "secretID")
	flag.StringVar(&secretKey, "secret-key", "[changeSecretKey]", "secretKey")
	flag.IntVar(&blockSize, "block-size", 1024*1024, "block size")
	flag.BoolVar(&debug, "debug", true, "debug oss")
}

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))
	flag.Parse()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	log := logger.ContextLog(ctx)

	// shutdown functions
	shutdownFunctions := make([]func(context.Context), 0)

	// signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	// init
	conf, err := initConf()
	server, err2 := initServer(conf)
	if err2 != nil {
		log.Fatalf("failed init file storage server, error: %v", err2.Error())
	}
	s, err := fsync.NewServer(ctx, path, server, fsync.OptionBufferSize(conf.BlockSize))
	if err != nil {
		log.Fatalf("failed to create server, error: %v", err.Error())
	}

	// errgroup
	g, ctx := errgroup.WithContext(ctx)
	g.Go(
		func() error {
			shutdownFunctions = append(
				shutdownFunctions, func(ctx context.Context) {
					s.Close()
				},
			)
			return err
		},
	)

	select {
	case <-ctx.Done():
		break
	case <-interrupt:
		break
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	for _, shutdown := range shutdownFunctions {
		shutdown(timeout)
	}
	err = g.Wait()
	if err != nil {
		panic(err)
	}

}

func initConf() (*Conf, error) {
	if confFile != "" {
		return NewConf(confFile)
	}
	conf := &Conf{
		Path:      path,
		Endpoint:  endpoint,
		Bucket:    bucket,
		SecretID:  secretID,
		SecretKey: secretKey,
		BlockSize: blockSize,
		Debug:     debug,
	}
	level := logger.InfoLevel
	if conf.Debug {
		level = logger.DebugLevel
	}
	err2 := logger.MinReportCallerLevel(level)
	if err2 != nil {
		logger.Log().Errorf("failed to init logger", err2.Error())
	}

	return conf, nil
}

func initServer(conf *Conf) (api.FileStorage, error) {
	oc := &oss2.Conf{
		Endpoint:  conf.Endpoint,
		Bucket:    conf.Bucket,
		SecretID:  conf.SecretID,
		SecretKey: conf.SecretKey,
	}
	s, err := oss2.NewOssStorage(
		oc,
		blockSize,
		debug,
	)

	return s, err
}
