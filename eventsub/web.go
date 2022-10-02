package eventsub

import (
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
	"superdicobot/internal/logger"
	"superdicobot/utils"
)

func Subscribe(notify chan string, config utils.ChannelConfig, allConfig utils.Config, Logger logger.LogWrapperObj) {

	Logger.Info("start new client", zap.String("channel", config.Channel))

	client, err := helix.NewClient(&helix.Options{
		ClientID:     allConfig.Webserver.Oauth.ClientId,
		ClientSecret: allConfig.Webserver.Oauth.ClientSecret,
	})
	p, err := client.RequestAppAccessToken([]string{})
	if err != nil {
		Logger.Error("unable to request app access Token", zap.Error(err))
	}

	client.SetAppAccessToken(p.Data.AccessToken)
	_, err = client.GetEventSubSubscriptions(&helix.EventSubSubscriptionsParams{
		//Status: helix.EventSubStatusEnabled, // This is optional.
	})
	if err != nil {
		Logger.Error("unable to request event sub subscriptions", zap.Error(err))
		// handle error
	}

	if err != nil {
		notify <- "erreur server: " + err.Error()
	}

	for _, event := range config.EventSub.Events {
		_, err = client.CreateEventSubSubscription(&helix.EventSubSubscription{
			Type:    event,
			Version: "1",
			Condition: helix.EventSubCondition{
				BroadcasterUserID: config.UserId,
			},
			Transport: helix.EventSubTransport{
				Method:   "webhook",
				Callback: allConfig.EventSub.Callback,
				Secret:   allConfig.EventSub.WebhookSecret,
			},
		})
		if err != nil {
			Logger.Error("unable to request create Sub", zap.Error(err))
			notify <- "erreur server: " + err.Error()
		}
	}

}
