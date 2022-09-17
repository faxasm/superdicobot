package main

import (
	"encoding/json"
	"fmt"
	"log"
	userpool "superdicobot/internal"
	"superdicobot/utils"

	"github.com/gempir/go-twitch-irc/v3"
)

type TimeoutPool map[string]*userpool.TTLMap

func main() {

	config, errConfig := utils.LoadConfig("./config")
	if errConfig != nil {
		log.Fatal("cannot load config:", errConfig)
	}
	client := twitch.NewClient(config.TwitchUser, config.TwitchOauth)
	channel := config.TwitchChannel
	timeoutPool := TimeoutPool{}
	timeoutPool[channel] = userpool.New(0, channel, client)
	t, _ := json.Marshal(client)

	// output conf client
	fmt.Println(string(t))

	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		//show bot status
		fmt.Println(message.Raw)
	})

	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		if message.BanDuration > 0 && message.BanDuration <= config.MaxTimeoutDuration {
			limit := message.Time.Unix() + int64(message.BanDuration)
			timeoutPool[message.Channel].Put(message.TargetUsername, message.TargetUserID, limit)
			fmt.Println("timeout detected for" + message.TargetUsername)
			fmt.Println(timeoutPool[channel].Display())
		}
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.Message == config.UntimeoutCmd {
			moderator, hasModerator := message.User.Badges["moderator"]
			broadcaster, hasBroadcaster := message.User.Badges["broadcaster"]
			if (hasModerator && moderator == 1) || (hasBroadcaster && broadcaster == 1) {
				if timeoutPool[message.Channel].Len() > 0 {
					println("untimeout detected")
					println(timeoutPool[message.Channel].Display())
					timeoutPool[message.Channel].UnTimeout()
				}
			}
		}
	})

	client.Join(channel)

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
