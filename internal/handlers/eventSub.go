package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"sync"
	"time"
)

type EventSubNotification struct {
	Subscription helix.EventSubSubscription `json:"subscription"`
	Challenge    string                     `json:"challenge"`
	Event        json.RawMessage            `json:"event"`
}

func EventCallback(c *gin.Context) {
	config := c.Value("config").(utils.Config)
	Logger := c.Value("logger").(logger.LogWrapperObj)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		Logger.Error("unable to get Body", zap.Error(err))
		return
	}
	defer c.Request.Body.Close()
	// verify that the notification came from twitch using the secret.
	if !helix.VerifyEventSubNotification(config.EventSub.WebhookSecret, c.Request.Header, string(body)) {
		Logger.Warn("no valid signature on subscription")
		return
	} else {
		Logger.Info("verified signature for subscription")
	}

	var vals EventSubNotification

	Logger.Info("get Event", zap.ByteString("event", body))
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&vals)
	if err != nil {
		Logger.Error("unable to decode msg", zap.Error(err))
		return
	}
	// if there's a challenge in the request, respond with only the challenge to verify your eventsub.
	if vals.Challenge != "" {
		c.Writer.Write([]byte(vals.Challenge))
		return
	}

	switch vals.Subscription.Type {
	case "channel.channel_points_custom_reward_redemption.add":
		ExecuteRewardRedemption(c, vals)
		return
	case "channel.channel_points_custom_reward_redemption.update":
		ExecuteRewardRedemptionUpdate(c, vals)
		return
	case "channel.ban":
		ExecuteBan(c, vals)
		return
	}
	//err = json.NewDecoder(bytes.NewReader(vals.Event)).Decode(&followEvent)

	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte("ok"))
}

func ExecuteRewardRedemption(c *gin.Context, notification EventSubNotification) {
	config := c.Value("config").(utils.Config)
	Logger := c.Value("logger").(logger.LogWrapperObj)

	var event helix.EventSubChannelPointsCustomRewardRedemptionEvent
	b, _ := json.Marshal(notification.Event)
	err := json.NewDecoder(bytes.NewReader(b)).Decode(&event)
	if err != nil {
		Logger.Error("unable to decode msg", zap.Error(err))
		return
	}
	// push to file
	channel := event.BroadcasterUserLogin
	reward := event.Reward.ID
	filePath := config.BddPath + "/events/" + channel + "/rewards/" + reward + ".csv"

	column := []string{
		channel,
		reward,
		event.Reward.Title,
		strconv.Itoa(event.Reward.Cost),
		event.UserID,
		event.UserLogin,
		event.UserName,
		event.RedeemedAt.Format(time.RFC3339),
		event.Status,
	}
	var m sync.Mutex
	m.Lock()
	if filePath != "" {
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				panic(err)
			}
			_, err := os.Create(filePath)
			if err != nil {
				panic(err)
			}
		}
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		Logger.Error("fail to open bdd", zap.Error(err))
	}
	w := csv.NewWriter(f)
	if err = w.Write(column); err != nil {
		Logger.Error("fail to write on bdd", zap.Error(err))
	}
	w.Flush()
	f.Close()
	m.Unlock()
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte("ok"))
}

func ExecuteRewardRedemptionUpdate(c *gin.Context, notification EventSubNotification) {
	config := c.Value("config").(utils.Config)
	Logger := c.Value("logger").(logger.LogWrapperObj)

	var event helix.EventSubChannelPointsCustomRewardRedemptionEvent
	b, _ := json.Marshal(notification.Event)
	err := json.NewDecoder(bytes.NewReader(b)).Decode(&event)
	if err != nil {
		Logger.Error("unable to decode msg", zap.Error(err))
		return
	}
	// push to file
	channel := event.BroadcasterUserLogin
	reward := event.Reward.ID
	filePath := config.BddPath + "/events/" + channel + "/rewards/" + reward + ".csv"

	column := []string{
		channel,
		reward,
		event.Reward.Title,
		strconv.Itoa(event.Reward.Cost),
		event.UserID,
		event.UserLogin,
		event.UserName,
		event.RedeemedAt.Format(time.RFC3339),
		event.Status,
		event.ID,
	}
	var m sync.Mutex
	m.Lock()
	if filePath != "" {
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				panic(err)
			}
			_, err := os.Create(filePath)
			if err != nil {
				panic(err)
			}
		}
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		Logger.Error("fail to open bdd", zap.Error(err))
	}
	w := csv.NewWriter(f)
	if err = w.Write(column); err != nil {
		Logger.Error("fail to write on bdd", zap.Error(err))
	}
	w.Flush()
	f.Close()
	m.Unlock()
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte("ok"))
}

func ExecuteBan(c *gin.Context, notification EventSubNotification) {

	// just log
	Logger := c.Value("logger").(logger.LogWrapperObj)

	b, _ := json.Marshal(notification)
	Logger.Info("ban event", zap.ByteString("event", b))
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte("ok"))
}
