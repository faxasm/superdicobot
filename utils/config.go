package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"superdicobot/internal/logger"
)

type Config struct {
	LoggerLevel string    `mapstructure:"loggerLevel"`
	LoggerFile  string    `mapstructure:"loggerFile"`
	BddPath     string    `mapstructure:"bddPath"`
	Bots        []Bot     `mapstructure:"bots"`
	EventSub    EventSub  `mapstructure:"eventSub"`
	Webserver   Webserver `mapstructure:"webserver"`
}

type Webserver struct {
	Oauth Oauth    `mapstructure:"oauth"`
	Hosts []string `mapstructure:"hosts"`
}

type Oauth struct {
	ClientId         string   `mapstructure:"clientId"`
	ClientSecret     string   `mapstructure:"clientSecret"`
	AppToken         string   `mapstructure:"appToken"`
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
	Channel     string                `mapstructure:"channel"`
	UserId      string                `mapstructure:"userId"`
	Token       string                `mapstructure:"token"`
	PingCmd     string                `mapstructure:"pingCmd"`
	LoggerLevel string                `mapstructure:"loggerLevel"`
	LoggerFile  string                `mapstructure:"loggerFile"`
	EventSub    EventSubChannelConfig `mapstructure:"eventSub"`
}

type EventSubChannelConfig struct {
	ClientId string   `mapstructure:"clientId"`
	Events   []string `mapstructure:"events"`
}

type EventSub struct {
	WebhookSecret string `mapstructure:"webhookSecret"`
	Callback      string `mapstructure:"callback"`
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

func ConfigMiddleware(config Config, logger logger.LogWrapperObj) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		c.Set("config", config)
		c.Set("logger", logger)

		if bot := c.Param("bot"); bot != "" {
			for _, botConfig := range config.Bots {
				if botConfig.User == bot {
					c.Set("botConfig", botConfig)
					if channel := c.Param("channel"); channel != "" {
						for _, channelConfig := range botConfig.Channels {
							if channelConfig.Channel == channel {
								c.Set("channelConfig", channelConfig)
							}
						}
					}
				}
			}
		}
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
