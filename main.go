package main

import (
	"context"
	"flag"
	"github.com/pjoc-team/fsync/internal/config"
	"github.com/pjoc-team/fsync/pkg/fsync"
	"github.com/pjoc-team/fsync/pkg/storage/api"
	oss2 "github.com/pjoc-team/fsync/pkg/storage/backend/oss"
	"github.com/pjoc-team/tracing/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"os"
	"os/signal"
	"strings"
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
	Path       string `yaml:"path" json:"path" xml:"path"`
	Endpoint   string `yaml:"endpoint" json:"endpoint" xml:"endpoint"`
	Bucket     string `yaml:"bucket" json:"bucket" xml:"bucket"`
	SecretID   string `yaml:"secretId" json:"secret_id" xml:"secret_id"`
	SecretKey  string `yaml:"secretKey" json:"secret_key" xml:"secret_key"`
	ConfPath   string `yaml:"confPath" json:"conf_path" xml:"conf_path"`
	BlockSize  int    `yaml:"blockSize" json:"block_size" xml:"block_size"`
	Debug      bool   `yaml:"debug" json:"debug" xml:"debug"`
	InitUpload bool   `yaml:"initUpload" json:"init_upload" xml:"init_upload"`
}

// Conf conf instance
var conf *Conf

func init() {
	log := logger.Log()
	conf = &Conf{}

	confVar := "conf"
	initUploadVar := "init-upload"
	pathVar := "path"
	confPathVar := "conf-path"
	endpointVar := "endpoint"
	bucketVar := "bucket"
	secretIDVar := "secret-id"
	secretKeyVar := "secret-key"
	blockSizeVar := "block-size"
	debugVar := "debug"

	pflag.StringVar(&confFile, confVar, "", "conf file path")
	pflag.BoolVar(&conf.InitUpload, initUploadVar, false, "upload all data when first time")
	pflag.StringVar(&conf.Path, pathVar, "./data/", "upload path")
	pflag.StringVar(&conf.ConfPath, confPathVar, "./conf/", "config path")
	pflag.StringVar(&conf.Endpoint, endpointVar, "https://cos.ap-guangzhou.myqcloud.com", "endpoint")
	pflag.StringVar(&conf.Bucket, bucketVar, "backup-1251070767", "bucket")
	pflag.StringVar(&conf.SecretID, secretIDVar, "[changeSecretID]", "secretID")
	pflag.StringVar(&conf.SecretKey, secretKeyVar, "[changeSecretKey]", "secretKey")
	pflag.IntVar(&conf.BlockSize, blockSizeVar, 1024*1024, "block size")
	pflag.BoolVar(&conf.Debug, debugVar, true, "debug oss")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err2 := viper.BindPFlags(pflag.CommandLine) // support env
	if err2 != nil {
		log.Fatal(err2.Error())
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	confFile = viper.GetString(confVar)
	conf.InitUpload = viper.GetBool(initUploadVar)
	conf.Path = viper.GetString(pathVar)
	conf.ConfPath = viper.GetString(confPathVar)
	conf.Endpoint = viper.GetString(endpointVar)
	conf.Bucket = viper.GetString(bucketVar)
	conf.SecretID = viper.GetString(secretIDVar)
	conf.SecretKey = viper.GetString(secretKeyVar)
	conf.BlockSize = viper.GetInt(blockSizeVar)
	conf.Debug = viper.GetBool(debugVar)

	log.Infof("get config: %#v", conf)
}

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))

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
		fsync.OptionInitUpload(conf.InitUpload),
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
		cs, err := config.NewConf(confFile)
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
	err2 = logger.SetLevel(level)
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
