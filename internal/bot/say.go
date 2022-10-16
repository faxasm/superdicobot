package bot

import (
	"encoding/csv"
	"fmt"
	"github.com/gempir/go-twitch-irc/v3"
	"go.uber.org/zap"
	"os"
	"sort"
	"strconv"
	"strings"
	"superdicobot/internal/bdd"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"sync"
	"time"
)

func SayScoreRewardCmd(rewardCmd bdd.RewardCmd, message twitch.PrivateMessage,
	channelInstance ChannelInstance, botConfig utils.Bot, allConfig utils.Config,
	client *twitch.Client, Logger logger.LogWrapperObj) bool {
	if StringInSlice(message.Message, rewardCmd.Aliases) {
		if isValidSender(message, botConfig.Administrator, rewardCmd.User) &&
			isNotInCoolDown(rewardCmd.CoolDown, rewardCmd.Aliases[0], channelInstance.MessageCoolDown) {
			channelInstance.Logger.Info("receive cmd", zap.Reflect("message", message))
			filePath := allConfig.BddPath + "/events/" + message.Channel + "/rewards/" + rewardCmd.Id + ".csv"
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
			if err := f.Close(); err != nil {
				Logger.Error("Unable to close input file "+filePath, zap.Error(err))
			}
			m.Unlock()
			if err != nil {
				Logger.Error("Unable to read input file "+filePath, zap.Error(err))
			}

			if len(records) > 0 {
				userScore := make(map[string]int)
				userName := make(map[string]string)
				for _, record := range records {
					user := record[4]
					name := record[6]
					userName[user] = name
					status := "fulfilled"
					if len(record) >= 9 {
						status = record[8]
					}
					if status != "fulfilled" {
						continue
					}
					if val, ok := userScore[user]; ok {
						userScore[user] = val + 1
						//do something here
					} else {
						userScore[user] = 1
					}
				}
				keys := make([]string, 0, len(userScore))

				for k := range userScore {
					keys = append(keys, k)
				}
				sort.SliceStable(keys, func(i, j int) bool {
					return userScore[keys[i]] > userScore[keys[j]]
				})
				msg := make([]string, 0, len(userScore))
				for _, k := range keys {
					msg = append(msg, fmt.Sprintf("%s %d", userName[k], userScore[k]))
				}
				client.Say(message.Channel, fmt.Sprintf(rewardCmd.Cmd, strings.Join(msg, " - ")))
				if rewardCmd.CoolDown != "" {
					if coolDown, err := strconv.Atoi(rewardCmd.CoolDown); err == nil {
						lastValid := time.Now().Add(time.Second * time.Duration(coolDown))
						channelInstance.MessageCoolDown.Put(rewardCmd.Aliases[0], "cooldown", lastValid.Unix())
					}
				}
			}
		}
		return true

	}
	return false
}

func SayLastRewardCmd(rewardCmd bdd.RewardCmd, message twitch.PrivateMessage,
	channelInstance ChannelInstance, botConfig utils.Bot, allConfig utils.Config,
	client *twitch.Client, Logger logger.LogWrapperObj) bool {
	if StringInSlice(message.Message, rewardCmd.Aliases) {
		if isValidSender(message, botConfig.Administrator, rewardCmd.User) &&
			isNotInCoolDown(rewardCmd.CoolDown, rewardCmd.Aliases[0], channelInstance.MessageCoolDown) {
			channelInstance.Logger.Info("receive last reward cmd", zap.Reflect("message", message))
			filePath := allConfig.BddPath + "/events/" + message.Channel + "/rewards/" + rewardCmd.Id + ".csv"
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
			if err := f.Close(); err != nil {
				Logger.Error("Unable to close input file "+filePath, zap.Error(err))
			}
			m.Unlock()
			if err != nil {
				Logger.Error("Unable to read input file "+filePath, zap.Error(err))
			}

			if len(records) > 0 {
				currentRecord := records[0]

				for _, record := range records {
					status := "unfulfilled"
					if len(record) >= 9 {
						status = record[8]
					}
					if status == "unfulfilled" {
						currentRecord = record
					}
				}
				name := currentRecord[6]
				client.Say(message.Channel, fmt.Sprintf(rewardCmd.Cmd, name))
				if rewardCmd.CoolDown != "" {
					if coolDown, err := strconv.Atoi(rewardCmd.CoolDown); err == nil {
						lastValid := time.Now().Add(time.Second * time.Duration(coolDown))
						channelInstance.MessageCoolDown.Put(rewardCmd.Aliases[0], "cooldown", lastValid.Unix())
					}
				}
			}
		}
		return true
	}
	return false
}

func SaySoldRewardCmd(rewardCmd bdd.RewardCmd, message twitch.PrivateMessage,
	channelInstance ChannelInstance, botConfig utils.Bot, allConfig utils.Config,
	client *twitch.Client, Logger logger.LogWrapperObj) bool {

	if StringInSlice(message.Message, rewardCmd.Aliases) {
		if isValidSender(message, botConfig.Administrator, rewardCmd.User) &&
			isNotInCoolDown(rewardCmd.CoolDown, rewardCmd.Aliases[0], channelInstance.MessageCoolDown) {
			channelInstance.Logger.Info("receive sold reward cmd", zap.Reflect("message", message))
			filePath := allConfig.BddPath + "/events/" + message.Channel + "/rewards/" + rewardCmd.Id + ".csv"
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
				userScore := 0
				for _, record := range records {
					status := "unfulfilled"
					if len(record) >= 9 {
						status = record[8]
					}
					if status == "unfulfilled" {
						userScore += 1
					} else {
						userScore -= 1
					}
				}
				userScore = userScore * rewardCmd.Unit
				score := strconv.Itoa(userScore)

				client.Say(message.Channel, fmt.Sprintf(rewardCmd.Cmd, score))
				if rewardCmd.CoolDown != "" {
					if coolDown, err := strconv.Atoi(rewardCmd.CoolDown); err == nil {
						lastValid := time.Now().Add(time.Second * time.Duration(coolDown))
						channelInstance.MessageCoolDown.Put(rewardCmd.Aliases[0], "cooldown", lastValid.Unix())
					}
				}
			}
		}
		return true
	}
	return false
}

func SayTotalRewardCmd(rewardCmd bdd.RewardCmd, message twitch.PrivateMessage,
	channelInstance ChannelInstance, botConfig utils.Bot, allConfig utils.Config,
	client *twitch.Client, Logger logger.LogWrapperObj) bool {

	if StringInSlice(message.Message, rewardCmd.Aliases) {
		if isValidSender(message, botConfig.Administrator, rewardCmd.User) &&
			isNotInCoolDown(rewardCmd.CoolDown, rewardCmd.Aliases[0], channelInstance.MessageCoolDown) {
			channelInstance.Logger.Info("receive sold reward cmd", zap.Reflect("message", message))
			filePath := allConfig.BddPath + "/events/" + message.Channel + "/rewards/" + rewardCmd.Id + ".csv"
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
				userScore := 0
				for _, record := range records {
					status := record[8]

					if status != "fulfilled" {
						continue
					}
					userScore += 1
				}
				userScore = userScore * rewardCmd.Unit
				score := strconv.Itoa(userScore)

				client.Say(message.Channel, fmt.Sprintf(rewardCmd.Cmd, score))
				if rewardCmd.CoolDown != "" {
					if coolDown, err := strconv.Atoi(rewardCmd.CoolDown); err == nil {
						lastValid := time.Now().Add(time.Second * time.Duration(coolDown))
						channelInstance.MessageCoolDown.Put(rewardCmd.Aliases[0], "cooldown", lastValid.Unix())
					}
				}
			}
		}
		return true
	}
	return false
}
