package bot

import (
	"encoding/csv"
	"github.com/go-co-op/gocron"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"superdicobot/internal/bdd"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"sync"
	"time"
)

type CInfo struct {
	start time.Time
	count int
}

func CronJob(allConfig utils.Config, botConfig utils.Bot, channelInstance ChannelInstance) func() {

	return func() {
		Logger := channelInstance.Logger

		hotConfig, err := bdd.GetBddConfig(allConfig, botConfig.User, channelInstance.ChannelConfig.Channel, Logger)
		if err != nil {
			Logger.Warn("unable to find bdd", zap.Error(err))
			return
		}
		if channelInstance.CronTask == nil {
			channelInstance.CronTask = &CronJobs{
				Scheduler:      gocron.NewScheduler(time.UTC),
				CronRewardCmds: &CronRewardJobs{},
			}
		}

		cronJobs := channelInstance.CronTask.CronRewardCmds
		if cronJobs.HasDiffCronRewardCmds(hotConfig.CronRewardCmds) {
			Logger.Info("have diff for cron", zap.Reflect("crons", hotConfig.CronRewardCmds))
			//rebootJobs()
			channelInstance.CronTask.Scheduler.Clear()
			if channelInstance.CronTask.Scheduler.IsRunning() {
				channelInstance.CronTask.Scheduler.Stop()
			}
			for _, newJob := range hotConfig.CronRewardCmds {
				if newJob.Period <= 0 {
					break
				}
				channelInstance.CronTask.Scheduler.Every(newJob.Period).Seconds().Do(
					func(newJob bdd.CronRewardCmd) {

						hotConfig, err := bdd.GetBddConfig(allConfig, botConfig.User, channelInstance.ChannelConfig.Channel, Logger)
						if err != nil {
							Logger.Warn("unable to find bdd", zap.Error(err))
							return
						}
						if !hotConfig.Activate {
							//deactivated bot
							return
						}

						channel := channelInstance.ChannelConfig.Channel
						if !*channelInstance.IsOnline {
							return
						}
						filePath := allConfig.BddPath + "/events/" + channel + "/rewards/" + newJob.Id + ".csv"
						var m sync.Mutex
						m.Lock()
						f, err := os.Open(filePath)
						if err != nil {
							Logger.Error("Unable to read input file "+filePath, zap.Error(err))
							if err := f.Close(); err != nil {
								Logger.Error("Unable to close input file "+filePath, zap.Error(err))
							}
						}

						csvReader := csv.NewReader(f)
						csvReader.FieldsPerRecord = -1
						records, err := csvReader.ReadAll()
						if err != nil {
							Logger.Error("Unable to read input file "+filePath, zap.Error(err))
						}
						if err := f.Close(); err != nil {
							Logger.Error("Unable to close input file "+filePath, zap.Error(err))
						}
						m.Unlock()
						if len(records) > 0 {
							userSold := 0
							userAction := 0
							for _, record := range records {
								status := "unfulfilled"
								if len(record) >= 9 {
									status = record[8]
								}

								dateCsv := record[7]
								dateAction, _ := time.Parse(time.RFC3339, dateCsv)

								loc, _ := time.LoadLocation("Europe/Paris")

								dateActionCest := dateAction.In(loc)
								currentDate := time.Now().In(loc)

								switch status {
								case "unfulfilled":
									userSold += 1
								case "fulfilled":
									userSold -= 1
									if dateActionCest.Format("2006-01-02") == currentDate.Format("2006-01-02") {
										userAction += 1
									}
								default:
									userSold -= 1
								}
							}
							userSold = userSold * newJob.Unit
							userAction = userAction * newJob.Unit
							if userSold > 0 && userAction > 0 {
								score := strconv.Itoa(userSold)
								action := strconv.Itoa(userAction)
								text := strings.Replace(newJob.SoldActionPositive, "{{solde}}", score, -1)
								text = strings.Replace(text, "{{action}}", action, -1)
								channelInstance.Client.Say(channel, text)
							} else if userSold > 0 {
								score := strconv.Itoa(userSold)
								text := strings.Replace(newJob.SoldPositive, "{{solde}}", score, -1)
								channelInstance.Client.Say(channel, text)
							} else if userAction > 0 {
								action := strconv.Itoa(userAction)
								text := strings.Replace(newJob.ActionPositive, "{{action}}", action, -1)
								channelInstance.Client.Say(channel, text)
							}
						}
					}, newJob)
			}
			channelInstance.CronTask.Scheduler.StartAsync()
			newCronRewardCmds := &CronRewardJobs{}
			*newCronRewardCmds = hotConfig.CronRewardCmds
			channelInstance.CronTask.CronRewardCmds = newCronRewardCmds
		}
	}
}

func CheckChannelStatus(allConfig utils.Config, botConfig utils.Bot, channelInstances map[string]ChannelInstance, Logger logger.LogWrapperObj) func() {
	apiClient, err := helix.NewClient(&helix.Options{
		ClientID:       allConfig.Webserver.Oauth.ClientId,
		ClientSecret:   allConfig.Webserver.Oauth.ClientSecret,
		AppAccessToken: allConfig.Webserver.Oauth.AppToken,
	})

	if err != nil {
		Logger.Error("client is OUT", zap.Error(err))
		panic("out apiClient is down" + err.Error())
	}

	channels := make([]string, 0)
	for ch, _ := range channelInstances {
		channels = append(channels, ch)
	}
	return func() {

		resp, err := apiClient.GetStreams(&helix.StreamsParams{
			UserLogins: channels,
			First:      len(channels),
		})
		if err != nil {
			Logger.Warn("cant check online channels")
			return
		}

		channelsOnline := make(map[string]CInfo, 0)
		for _, channelResponse := range resp.Data.Streams {
			channelsOnline[channelResponse.UserLogin] = CInfo{
				start: channelResponse.StartedAt,
				count: channelResponse.ViewerCount,
			}
		}

		for _, channelInstance := range channelInstances {
			cInfo, isOnline := channelsOnline[channelInstance.ChannelConfig.Channel]
			Logger.Info("startedAt", zap.String("channel", channelInstance.ChannelConfig.Channel),
				zap.Time("started", cInfo.start),
				zap.Int("viewers", cInfo.count))
			if channelInstance.IsOnline != nil {
				*channelInstance.IsOnline = isOnline
				if isOnline {
					*channelInstance.StartedAt = cInfo.start
					*channelInstance.Viewers = cInfo.count
				}
			} else {
				channelInstance.IsOnline = &isOnline
				if isOnline {
					channelInstance.StartedAt = &cInfo.start
					channelInstance.Viewers = &cInfo.count

				}
			}
		}

	}
}
