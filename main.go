package main

import (
	"context"
	"flag"
	"github.com/pjoc-team/fsync/internal/viper"
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
	confFile string
)

const (
	threadPoolSize = 16
)

// Conf config struct
type Conf struct {
	Path      string
	Endpoint  string
	Bucket    string
	SecretID  string
	SecretKey string
	ConfPath  string
	BlockSize int
	Debug     bool
}

// Conf conf instance
var conf *Conf

func init() {
	conf = &Conf{}

	flag.StringVar(&confFile, "conf", "", "conf file path")
	flag.StringVar(&conf.Path, "path", "./data/", "upload path")
	flag.StringVar(&conf.ConfPath, "conf-path", "./conf/", "config path")
	flag.StringVar(&conf.Endpoint, "endpoint", "https://cos.ap-guangzhou.myqcloud.com", "endpoint")
	flag.StringVar(&conf.Bucket, "bucket", "backup-1251070767", "bucket")
	flag.StringVar(&conf.SecretID, "secret-id", "[changeSecretID]", "secretID")
	flag.StringVar(&conf.SecretKey, "secret-key", "[changeSecretKey]", "secretKey")
	flag.IntVar(&conf.BlockSize, "block-size", 1024*1024, "block size")
	flag.BoolVar(&conf.Debug, "debug", true, "debug oss")
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
	if err != nil {
		log.Fatalf("failed init file storage server, error: %v", err.Error())
	}
	server, err := initServer(conf)
	if err != nil {
		log.Fatalf("failed init file storage server, error: %v", err.Error())
	}
	s, err := fsync.NewServer(
		ctx, conf.Path, server, fsync.OptionBufferSize(conf.BlockSize),
		fsync.OptionConfPath(conf.ConfPath), fsync.OptionThreadPoolSize(threadPoolSize),
	)
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
		conf = &Conf{}
		cs, err := viper.NewConf(confFile)
		if err != nil {
			logger.Log().Errorf("failed to init conf", err.Error())
			return nil, err
		}
		err = cs.UnmarshalConfig(conf)
		if err != nil {
			logger.Log().Errorf("failed to init conf", err.Error())
			return nil, err
		}
		return conf, nil
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
		conf.BlockSize,
		conf.Debug,
	)

	return s, err
}
