package main

import (
	"encoding/json"
	"fmt"
	"go.uber.org/ratelimit"
	"log"
	"strings"
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
	//rateLimiter Global
	rl := ratelimit.New(3) // per second limit twitch:  100 per 30 seconds

	client := twitch.NewClient(config.TwitchUser, config.TwitchOauth)
	configChannel := config.TwitchChannel
	channels := strings.Split(configChannel, ",")

	timeoutPool := TimeoutPool{}

	for _, channel := range channels {
		timeoutPool[channel] = userpool.New(0, channel, client, rl)
	}
	t, _ := json.Marshal(client)

	// output conf client
	fmt.Println(string(t))

	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		//show bot status
		fmt.Println(message.Raw)
	})

	client.OnPongMessage(func(message twitch.PongMessage) {
		//show pong bot status
		fmt.Println(message.Raw)
	})

	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		if message.BanDuration > 0 && message.BanDuration <= config.MaxTimeoutDuration {
			limit := message.Time.Unix() + int64(message.BanDuration)
			timeoutPool[message.Channel].Put(message.TargetUsername, message.TargetUserID, limit)
			fmt.Println("timeout detected for" + message.TargetUsername)
			fmt.Println(timeoutPool[message.Channel].Display())
		}
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		moderator, hasModerator := message.User.Badges["moderator"]
		broadcaster, hasBroadcaster := message.User.Badges["broadcaster"]
		if (hasModerator && moderator == 1) || (hasBroadcaster && broadcaster == 1) {

			if message.Message == config.PingCmd {
				println("receive ping from: " + message.Channel + " by: " + message.User.Name)
				rl.Take()
				client.Say(message.Channel, "Pong ! @"+message.User.Name)
			}

			if message.Message == config.UntimeoutCmd {
				if timeoutPool[message.Channel].Len() > 0 {
					println("untimeout detected")
					println(timeoutPool[message.Channel].Display())
					timeoutPool[message.Channel].UnTimeout()
				}
			}
		}
	})

	fmt.Println("Start listening on: " + configChannel)
	client.Join(channels...)

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
