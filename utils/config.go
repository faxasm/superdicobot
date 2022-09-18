package utils

import (
	"github.com/spf13/viper"
)

type Config struct {
	TwitchUser         string `mapstructure:"TWITCH_USER"`
	TwitchOauth        string `mapstructure:"TWITCH_OAUTH"`
	TwitchChannel      string `mapstructure:"TWITCH_CHANNEL"`
	UntimeoutCmd       string `mapstructure:"UNTIMEOUT_CMD"`
	PingCmd            string `mapstructure:"PING_CMD"`
	MaxTimeoutDuration int    `mapstructure:"MAX_TIMEOUT_DURATION"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
