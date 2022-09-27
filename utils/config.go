package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Config struct {
	LoggerLevel string    `mapstructure:"loggerLevel"`
	LoggerFile  string    `mapstructure:"loggerFile"`
	Bots        []Bot     `mapstructure:"bots"`
	Webserver   Webserver `mapstructure:"webserver"`
}

type Webserver struct {
	Oauth Oauth    `mapstructure:"oauth"`
	Hosts []string `mapstructure:"hosts"`
}

type Oauth struct {
	ClientId         string   `mapstructure:"clientId"`
	ClientSecret     string   `mapstructure:"clientSecret"`
	Scopes           []string `mapstructure:"scopes"`
	RedirectURL      string   `mapstructure:"redirectUrl"`
	CookieSecret     string   `mapstructure:"cookieSecret"`
	StateCallbackKey string   `mapstructure:"stateCallbackKey"`
	OauthSessionName string   `mapstructure:"oauthSessionName"`
	OauthTokenKey    string   `mapstructure:"oauthTokenKey"`
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

func ConfigMiddleware(config Config) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		c.Set("config", config)
		c.Next()
	}
}

func GetSafeConfig(config Config, user string) Config {
	newConfig := Config{}

	for _, bot := range config.Bots {
		newBot := Bot{}
		newBot.User = bot.User
		newBot.Administrator = bot.Administrator
		newBot.Channels = []ChannelConfig{}

		for _, channel := range bot.Channels {
			if channel.Channel == user || bot.Administrator == user {
				newBot.Channels = append(newBot.Channels, channel)
			}
		}
		if len(newBot.Channels) > 0 {
			newConfig.Bots = append(newConfig.Bots, newBot)
		}
	}

	return newConfig

}

func GetBot(config Config, botName string) (Bot, error) {
	for _, bot := range config.Bots {
		if bot.User == botName {
			return bot, nil
		}
	}

	return Bot{}, errors.New("unable to find bot")
}
