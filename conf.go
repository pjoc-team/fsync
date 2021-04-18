package main

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pjoc-team/tracing/logger"
	"github.com/spf13/viper"
	"strings"
)

// NewConf parse conf
func NewConf(filePath string) (*Conf, error) {
	v := viper.New()
	v.SetConfigFile(filePath)
	configType := configType(filePath)
	if configType == "" {
		return nil, fmt.Errorf("unknown file extension of file: %v", filePath)
	}
	v.SetConfigType(configType)
	options := viperDecoderConfig(configType)
	conf := &Conf{}
	err := v.Unmarshal(conf, options...)
	if err != nil {
		logger.Log().Errorf("failed to unmarshal file: %v, error: %v", filePath, err.Error())
		return nil, err
	}
	return conf, nil
}

func viperDecoderConfig(configType string) []viper.DecoderConfigOption {
	switch configType {
	case "yaml", "yml":
		return []viper.DecoderConfigOption{
			func(config *mapstructure.DecoderConfig) {
				config.TagName = "yaml"
			},
		}
	case "json", "toml", "properties", "props", "prop", "hcl", "dotenv", "env", "ini":
		return []viper.DecoderConfigOption{
			func(config *mapstructure.DecoderConfig) {
				config.TagName = configType
			},
		}
	}
	return nil
}

func configType(filePath string) string {
	fileIndex := strings.LastIndex(filePath, "/")
	if fileIndex >= 0 {
		filePath = filePath[fileIndex:]
	}
	index := strings.LastIndex(filePath, ".")
	if index < 0 {
		return ""
	}
	return filePath[index+1:]
}
