package utils

import (
	"github.com/spf13/viper"
)

type Config struct {
	LoggerLevel string `mapstructure:"loggerLevel"`
	LoggerFile  string `mapstructure:"loggerFile"`
	Bots        []Bot  `mapstructure:"bots"`
}

type Bot struct {
	User          string          `mapstructure:"user"`
	Oauth         string          `mapstructure:"oauth"`
	LoggerLevel   string          `mapstructure:"loggerLevel"`
	LoggerFile    string          `mapstructure:"loggerFile"`
	Administrator string          `mapstructure:"administrator"`
	Channels      []ChannelConfig `mapstructure:"channels"`
}

type ChannelConfig struct {
	Channel            string `mapstructure:"channel"`
	UnTimeoutCmd       string `mapstructure:"unTimeoutCmd"`
	PingCmd            string `mapstructure:"pingCmd"`
	MaxTimeoutDuration int    `mapstructure:"maxTimeoutDuration"`
	LoggerLevel        string `mapstructure:"loggerLevel"`
	LoggerFile         string `mapstructure:"loggerFile"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
