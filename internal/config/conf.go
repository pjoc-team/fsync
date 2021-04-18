package config

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pjoc-team/tracing/logger"
	"github.com/spf13/viper"
	"strings"
)

// Config config
type Config struct {
	v       *viper.Viper
	options []viper.DecoderConfigOption
}

// NewConf parse conf
func NewConf(filePath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(filePath)
	configType := configType(filePath)
	if configType == "" {
		return nil, fmt.Errorf("unknown file extension of file: %v", filePath)
	}
	v.SetConfigType(configType)
	c := &Config{}

	options := DecoderConfig(configType)
	c.options = options
	c.v = v
	return c, nil
}

// UnmarshalConfig unmarshal config to ptr
func (c *Config) UnmarshalConfig(conf interface{}) error {
	err := c.v.Unmarshal(conf, c.options...)
	if err != nil {
		logger.Log().Errorf("failed to unmarshal conf, error: %v", err.Error())
		return err
	}
	return nil
}

// DecoderConfig decoder config
func DecoderConfig(configType string) []viper.DecoderConfigOption {
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
