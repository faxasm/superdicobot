package bot

import (
	"github.com/gempir/go-twitch-irc/v3"
	"go.uber.org/zap"
	"strings"
	userpool "superdicobot/internal"
	"superdicobot/internal/logger"
	"superdicobot/utils"
)

type ChannelInstance struct {
	ChannelConfig *utils.ChannelConfig
	Logger        logger.LogWrapperObj
	TimeoutPool   *userpool.TTLMap
}

func NewChannelInstance(config utils.ChannelConfig, client *twitch.Client) ChannelInstance {

	variableConfig := &utils.ChannelConfig{
		Channel:            config.Channel,
		UnTimeoutCmd:       config.UnTimeoutCmd,
		PingCmd:            config.PingCmd,
		MaxTimeoutDuration: config.MaxTimeoutDuration,
		LoggerLevel:        config.LoggerLevel,
	}

	Logger := logger.NewLogger(variableConfig.LoggerLevel, config.LoggerFile)
	TimeoutPool := userpool.New(0, variableConfig.Channel, client)

	return ChannelInstance{
		ChannelConfig: variableConfig,
		Logger:        Logger,
		TimeoutPool:   TimeoutPool,
	}
}

func NewBot(notify chan string, botConfig utils.Bot) {
	// output conf client

	client := twitch.NewClient(botConfig.User, botConfig.Oauth)

	Logger := logger.NewLogger(botConfig.LoggerLevel, botConfig.LoggerFile)

	Logger.Info("bot client", zap.Reflect("client", client))

	var channels = make([]string, 0)

	channelInstances := make(map[string]ChannelInstance, len(botConfig.Channels))
	for _, channelConfig := range botConfig.Channels {
		channelInstances[channelConfig.Channel] = NewChannelInstance(channelConfig, client)
		channels = append(channels, channelConfig.Channel)
	}

	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		//show bot status
		Logger.Info("response.OnGlobalUserStateMessage", zap.Reflect("message", message))
	})

	client.OnPongMessage(func(message twitch.PongMessage) {
		//show pong bot status
		Logger.Debug("response.OnPongMessage", zap.Reflect("message", message))
	})

	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {

		channelInstance := channelInstances[message.Channel]
		if message.BanDuration > 0 && message.BanDuration <= channelInstance.ChannelConfig.MaxTimeoutDuration {
			limit := message.Time.Unix() + int64(message.BanDuration)
			channelInstance.TimeoutPool.Put(message.TargetUsername, message.TargetUserID, limit)
			channelInstance.Logger.Info("timeout detected for", zap.Reflect("message", message))
			channelInstance.Logger.Info("current timeoutPool",
				zap.String("channel", message.Channel),
				zap.String("pool", channelInstance.TimeoutPool.Display()))
		}
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if isValidSender(message, botConfig.Administrator) {
			channelInstance := channelInstances[message.Channel]
			if message.Message == channelInstance.ChannelConfig.PingCmd {
				channelInstance.Logger.Info("receive ping", zap.Reflect("message", message))
				client.Reply(message.Channel, message.ID, "Pong !")
			}

			if message.Message == channelInstance.ChannelConfig.UnTimeoutCmd {
				if channelInstance.TimeoutPool.Len() > 0 {
					channelInstance.Logger.Info("Untimeout detected",
						zap.String("channel", message.Channel),
						zap.Reflect("message", message),
						zap.String("pool", channelInstance.TimeoutPool.Display()))
					channelInstance.TimeoutPool.UnTimeout()
				}
			}
		}
	})

	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		channelInstance := channelInstances[message.Channel]
		channelInstance.Logger.Info("Notice Message detected",
			zap.String("channel", message.Channel),
			zap.Reflect("message", message))
	})

	Logger.Info("Start listening on", zap.String("channel", strings.Join(channels, ",")))
	client.Join(channels...)
	err := client.Connect()
	if err != nil {
		notify <- "panic" + botConfig.User + ": " + err.Error()
		panic(err)
	}
}

func isValidSender(message twitch.PrivateMessage, administrator string) bool {
	moderator, hasModerator := message.User.Badges["moderator"]
	broadcaster, hasBroadcaster := message.User.Badges["broadcaster"]
	return (hasModerator && moderator == 1) || (hasBroadcaster && broadcaster == 1) || message.User.Name == administrator
}
