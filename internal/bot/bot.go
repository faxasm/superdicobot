package bot

import (
	"encoding/csv"
	"fmt"
	"github.com/gempir/go-twitch-irc/v3"
	"github.com/go-co-op/gocron"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"superdicobot/eventsub"
	userpool "superdicobot/internal"
	"superdicobot/internal/bdd"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"sync"
	"time"
)

type ChannelInstance struct {
	ChannelConfig   *utils.ChannelConfig
	AllConfig       utils.Config
	Logger          logger.LogWrapperObj
	TimeoutPool     *userpool.TTLMap
	MessageCoolDown *userpool.TTLCmdMap
	Client          *twitch.Client
	CronTask        *CronJobs
	IsOnline        *bool
	StartedAt       *time.Time
	ChannelClient   *helix.Client
	Viewers         *int
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

func NewChannelInstance(config utils.ChannelConfig, client *twitch.Client, allConfig utils.Config) ChannelInstance {

	variableConfig := &utils.ChannelConfig{
		Channel:     config.Channel,
		PingCmd:     config.PingCmd,
		LoggerLevel: config.LoggerLevel,
	}
	Logger := logger.NewLogger(variableConfig.LoggerLevel, config.LoggerFile)
	TimeoutPool := userpool.New(0, variableConfig.Channel, client)
	messageCoolDown := userpool.NewCmdPool(0)

	var channelApiClient *helix.Client
	if config.Token != "" {
		var err error
		channelApiClient, err = helix.NewClient(&helix.Options{
			ClientID:        allConfig.Webserver.Oauth.ClientId,
			ClientSecret:    allConfig.Webserver.Oauth.ClientSecret,
			AppAccessToken:  allConfig.Webserver.Oauth.AppToken,
			UserAccessToken: config.Token,
		})
		if err != nil {
			panic(err)
		}
	}
	isOnline := false
	StartedAt := time.Now()
	viewers := 0

	newConfig := config

	return ChannelInstance{
		ChannelConfig:   &newConfig,
		AllConfig:       allConfig,
		Logger:          Logger,
		TimeoutPool:     TimeoutPool,
		MessageCoolDown: messageCoolDown,
		Client:          client,
		IsOnline:        &isOnline,
		ChannelClient:   channelApiClient,
		StartedAt:       &StartedAt,
		Viewers:         &viewers,
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
		channelInstances[channelConfig.Channel] = NewChannelInstance(channelConfig, client, allConfig)
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

		for _, rewardCmd := range hotConfig.TotalRewardCmds {
			if SayTotalRewardCmd(rewardCmd, message, channelInstance, botConfig, allConfig, client, Logger) {
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
				if strings.Contains(cmd, "{{subCount}}") {
					cmd = strings.Replace(cmd, "{{subCount}}", channelInstance.getSubCount(), -1)
				}

				if strings.Contains(cmd, "{{followerCount}}") {
					cmd = strings.Replace(cmd, "{{followerCount}}", channelInstance.getFollowerCount(), -1)
				}

				if strings.Contains(cmd, "{{viewerCount}}") {
					cmd = strings.Replace(cmd, "{{viewerCount}}", channelInstance.getViewersCount(), -1)
				}

				if strings.Contains(cmd, "{{streamDuration}}") {
					start := channelInstance.getStreamStartedAt()
					distStarted := int64(0)
					if start != nil {
						distStarted = time.Now().Unix() - start.Unix()
					}
					startStream, _ := time.ParseDuration(fmt.Sprintf("%ds", distStarted))

					cmd = strings.Replace(cmd, "{{streamDuration}}", fmt.Sprintf("%s", startStream), -1)
				}

				if strings.Contains(cmd, "{{lastSubDuration}}") {
					start := channelInstance.getLastSubDateMessage()
					distStarted := int64(0)
					distStarted = time.Now().Unix() - start.Unix()
					startStream, _ := time.ParseDuration(fmt.Sprintf("%ds", distStarted))

					cmd = strings.Replace(cmd, "{{lastSubDuration}}", fmt.Sprintf("%s", startStream), -1)
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

			if message.Message == "!testsub" {
				start := channelInstance.getStreamStartedAt()
				now := time.Now()
				distLastSub := now.Unix() - channelInstance.getLastSubDateMessage().Unix()
				distStarted := int64(0)
				if start != nil {
					distStarted = now.Unix() - start.Unix()
				}

				startStream, _ := time.ParseDuration(fmt.Sprintf("%ds", distStarted))
				lastSub, _ := time.ParseDuration(fmt.Sprintf("%ds", distLastSub))

				msg := "Dico has been going for %s there are %s viewers, %s followers and %s subscribers. It has been %s since anyone subbed."
				client.Say(message.Channel, fmt.Sprintf(msg, startStream, channelInstance.getViewersCount(), channelInstance.getFollowerCount(), channelInstance.getSubCount(), lastSub))
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

func (channelInstance ChannelInstance) getStreamStartedAt() *time.Time {
	if channelInstance.IsOnline != nil && *channelInstance.IsOnline {
		return channelInstance.StartedAt
	}
	return nil
}

func (channelInstance ChannelInstance) getViewersCount() string {
	if channelInstance.Viewers == nil {
		return "0"
	}
	return strconv.Itoa(*channelInstance.Viewers)
}

func (channelInstance ChannelInstance) getSubCount() string {
	if channelInstance.ChannelClient == nil {
		return ""
	}
	subs, err := channelInstance.ChannelClient.GetSubscriptions(&helix.SubscriptionsParams{
		BroadcasterID: channelInstance.ChannelConfig.UserId,
		First:         1,
	})

	if err != nil {
		channelInstance.Logger.Warn("unable to fetch subs", zap.Error(err))
		return ""
	}
	return strconv.Itoa(subs.Data.Total)
}

func (channelInstance ChannelInstance) getFollowerCount() string {
	if channelInstance.ChannelClient == nil {
		return ""
	}
	channelInstance.Logger.Info("conf", zap.Reflect("conf", channelInstance.ChannelConfig))
	subs, err := channelInstance.ChannelClient.GetUsersFollows(&helix.UsersFollowsParams{
		First: 1,
		ToID:  channelInstance.ChannelConfig.UserId,
	})

	if err != nil {
		channelInstance.Logger.Warn("unable to fetch subs", zap.Error(err))
		return ""
	}
	channelInstance.Logger.Warn("unable to fetch subs", zap.Reflect("data", subs))

	return strconv.Itoa(subs.Data.Total)
}

func (channelInstance ChannelInstance) getLastSubDateMessage() time.Time {
	var lastDate time.Time

	filePath := channelInstance.AllConfig.BddPath + "/events/" + channelInstance.ChannelConfig.Channel + "/subs/messages.csv"
	var m sync.Mutex
	m.Lock()
	f, err := os.Open(filePath)
	if err != nil {
		channelInstance.Logger.Error("Unable to read input file "+filePath, zap.Error(err))
		if err := f.Close(); err != nil {
			channelInstance.Logger.Error("Unable to close input file "+filePath, zap.Error(err))
		}
	}

	csvReader := csv.NewReader(f)
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		channelInstance.Logger.Error("Unable to read input file "+filePath, zap.Error(err))
	}
	if err := f.Close(); err != nil {
		channelInstance.Logger.Error("Unable to close input file "+filePath, zap.Error(err))
	}
	m.Unlock()
	if len(records) > 0 {
		last := records[len(records)-1]
		date, _ := time.Parse(time.RFC3339, last[8])
		lastDate = date
	}

	filePathGifts := channelInstance.AllConfig.BddPath + "/events/" + channelInstance.ChannelConfig.Channel + "/subs/gifts.csv"
	m.Lock()
	fGifts, err := os.Open(filePathGifts)
	if err != nil {
		channelInstance.Logger.Error("Unable to read input file "+filePathGifts, zap.Error(err))
		if err := f.Close(); err != nil {
			channelInstance.Logger.Error("Unable to close input file "+filePathGifts, zap.Error(err))
		}
	}

	csvReaderGifts := csv.NewReader(fGifts)
	csvReaderGifts.FieldsPerRecord = -1
	recordsGifts, err := csvReaderGifts.ReadAll()
	if err != nil {
		channelInstance.Logger.Error("Unable to read input file "+filePathGifts, zap.Error(err))
	}
	if err := f.Close(); err != nil {
		channelInstance.Logger.Error("Unable to close input file "+filePathGifts, zap.Error(err))
	}
	m.Unlock()
	if len(recordsGifts) > 0 {
		last := recordsGifts[len(recordsGifts)-1]
		date, _ := time.Parse(time.RFC3339, last[8])
		if lastDate.Before(date) {
			lastDate = date
		}
	}

	return lastDate
}
