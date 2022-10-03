package bot

import (
	"fmt"
	"github.com/gempir/go-twitch-irc/v3"
	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"superdicobot/eventsub"
	userpool "superdicobot/internal"
	"superdicobot/internal/bdd"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"time"
)

type ChannelInstance struct {
	ChannelConfig   *utils.ChannelConfig
	Logger          logger.LogWrapperObj
	TimeoutPool     *userpool.TTLMap
	MessageCoolDown *userpool.TTLCmdMap
	Client          *twitch.Client
	CronTask        *CronJobs
	IsOnline        *bool
}
type CronRewardJobs []bdd.CronRewardCmd

type CronJobs struct {
	Scheduler      *gocron.Scheduler
	CronRewardCmds *CronRewardJobs
}

func (currentJobs *CronRewardJobs) HasDiffCronRewardCmds(cronJobs []bdd.CronRewardCmd) bool {
	if currentJobs == nil && len(cronJobs) > 0 {
		return true
	}
	if currentJobs == nil {
		return false
	}
	if len(cronJobs) != len(*currentJobs) {
		return true
	}
	for i, job := range *currentJobs {
		if job != cronJobs[i] {
			return true
		}
	}
	return false
}

func NewChannelInstance(config utils.ChannelConfig, client *twitch.Client) ChannelInstance {

	variableConfig := &utils.ChannelConfig{
		Channel:     config.Channel,
		PingCmd:     config.PingCmd,
		LoggerLevel: config.LoggerLevel,
	}
	Logger := logger.NewLogger(variableConfig.LoggerLevel, config.LoggerFile)
	TimeoutPool := userpool.New(0, variableConfig.Channel, client)
	messageCoolDown := userpool.NewCmdPool(0)

	isOnline := false
	return ChannelInstance{
		ChannelConfig:   variableConfig,
		Logger:          Logger,
		TimeoutPool:     TimeoutPool,
		MessageCoolDown: messageCoolDown,
		Client:          client,
		IsOnline:        &isOnline,
	}
}

func NewBot(notify chan string, botConfig utils.Bot, allConfig utils.Config) {
	// output conf client

	client := twitch.NewClient(botConfig.User, botConfig.Oauth)

	Logger := logger.NewLogger(botConfig.LoggerLevel, botConfig.LoggerFile)

	Logger.Info("bot client", zap.Reflect("client", client))

	var channels = make([]string, 0)

	channelInstances := make(map[string]ChannelInstance, len(botConfig.Channels))
	for _, channelConfig := range botConfig.Channels {
		channelInstances[channelConfig.Channel] = NewChannelInstance(channelConfig, client)
		channels = append(channels, channelConfig.Channel)
		Logger.Info("start", zap.String("config channel", channelConfig.Channel))
		if channelConfig.EventSub.ClientId != "" {
			eventsub.Subscribe(notify, channelConfig, allConfig, Logger)
		}

		//cron job that check every 30 sec for new or modified cron
		s := gocron.NewScheduler(time.UTC)
		if _, err := s.Every(30).Seconds().Do(CronJob(allConfig, botConfig, channelInstances[channelConfig.Channel])); err != nil {
			Logger.Warn("unable to init sync", zap.Error(err))
		}
		s.StartAsync()
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
		hotConfig, err := bdd.GetBddConfig(allConfig, botConfig.User, message.Channel, Logger)
		if err != nil {
			Logger.Warn("unable to find bdd", zap.Error(err))
			return
		}
		if message.BanDuration > 0 && message.BanDuration <= hotConfig.UnTimeoutCmd.MaxTimeoutDuration {
			limit := message.Time.Unix() + int64(message.BanDuration)
			channelInstance.TimeoutPool.Put(message.TargetUsername, message.TargetUserID, limit)
			channelInstance.Logger.Info("timeout detected for", zap.Reflect("message", message))
			channelInstance.Logger.Info("current timeoutPool",
				zap.String("channel", message.Channel),
				zap.String("pool", channelInstance.TimeoutPool.Display()))
		}
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.User.Name == botConfig.User {
			return
		}

		channelInstance := channelInstances[message.Channel]
		//custom cmd
		hotConfig, err := bdd.GetBddConfig(allConfig, botConfig.User, message.Channel, Logger)
		if err != nil {
			Logger.Warn("unable to find bdd", zap.Error(err))
			return
		}

		if !hotConfig.Activate {
			Logger.Info("bot deactivated")
			return
		}
		for _, rewardCmd := range hotConfig.RewardCmds {
			if SayScoreRewardCmd(rewardCmd, message, channelInstance, botConfig, allConfig, client, Logger) {
				break
			}
		}

		for _, rewardCmd := range hotConfig.LastRewardCmds {
			if SayLastRewardCmd(rewardCmd, message, channelInstance, botConfig, allConfig, client, Logger) {
				break
			}
		}

		for _, rewardCmd := range hotConfig.SoldRewardCmds {
			if SaySoldRewardCmd(rewardCmd, message, channelInstance, botConfig, allConfig, client, Logger) {
				break
			}
		}

		for _, customCmd := range hotConfig.CustomCmds {
			startWith, endOfMatch := stringStartWithWord(message.Message, customCmd.Aliases)
			if startWith && isValidSender(message, botConfig.Administrator, customCmd.User) &&
				isNotInCoolDown(customCmd.CoolDown, customCmd.Aliases[0], channelInstance.MessageCoolDown) {

				channelInstance.Logger.Info("command detected",
					zap.String("channel", message.Channel),
					zap.Reflect("message", message))

				cmd := customCmd.Cmd
				if strings.Contains(customCmd.Cmd, "%s") {
					cmd = fmt.Sprintf(customCmd.Cmd, endOfMatch)
				}
				client.Say(message.Channel, cmd)
				if customCmd.CoolDown != "" {
					if coolDown, err := strconv.Atoi(customCmd.CoolDown); err == nil {
						lastValid := time.Now().Add(time.Second * time.Duration(coolDown))
						channelInstance.MessageCoolDown.Put(customCmd.Aliases[0], "cooldown", lastValid.Unix())
					}
				}
			}
		}

		if isValidSender(message, botConfig.Administrator, "moderator") {
			if message.Message == channelInstance.ChannelConfig.PingCmd {
				channelInstance.Logger.Info("receive ping", zap.Reflect("message", message))
				client.Reply(message.Channel, message.ID, "Pong !")
			}

			if message.Message == hotConfig.UnTimeoutCmd.Cmd {
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

	Logger.Info("Start Check Online Channels")
	//cron job that check every 60 sec for Check Online
	s := gocron.NewScheduler(time.UTC)
	if _, err := s.Every(60).Seconds().Do(CheckChannelStatus(allConfig, botConfig, channelInstances, Logger)); err != nil {
		Logger.Warn("unable to init sync", zap.Error(err))
	}
	s.StartAsync()

	Logger.Info("Start listening on", zap.String("channel", strings.Join(channels, ",")))
	client.Join(channels...)
	err := client.Connect()
	if err != nil {
		notify <- "panic" + botConfig.User + ": " + err.Error()
		panic(err)
	}
}
func isNotInCoolDown(currentCoolDown string, alias string, cmdPool *userpool.TTLCmdMap) bool {

	if currentCoolDown == "" {
		return true
	}

	if _, err := strconv.Atoi(currentCoolDown); err != nil {
		return true
	}

	if cmdPool.Get(alias) != "" {
		return false
	}

	return true
}

func isValidSender(message twitch.PrivateMessage, administrator string, mode string) bool {
	moderator, hasModerator := message.User.Badges["moderator"]
	broadcaster, hasBroadcaster := message.User.Badges["broadcaster"]

	switch mode {
	case "all":
		return true
	case "moderator":
		return (hasModerator && moderator == 1) || (hasBroadcaster && broadcaster == 1) || message.User.Name == administrator
	case "streamer":
		return (hasBroadcaster && broadcaster == 1) || message.User.Name == administrator
	}
	return false
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func stringStartWithWord(a string, list []string) (bool, string) {
	for _, b := range list {

		if a == b || strings.HasPrefix(a, b+" ") {
			endOfMatch := strings.TrimPrefix(a, b)
			return true, strings.TrimSpace(endOfMatch)
		}
	}
	return false, ""
}
