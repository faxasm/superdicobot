package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"sync"
	"time"
)

type ChessEvent struct {
	White      string    `json:"white"`
	WhiteClock string    `json:"whiteClock"`
	Black      string    `json:"black"`
	BlackClock string    `json:"blackClock"`
	Turn       string    `json:"turn"`
	Speed      string    `json:"speed"`
	ApiKey     string    `json:"apiKey"`
	Channel    string    `json:"channel"`
	BotName    string    `json:"botName"`
	User       string    `json:"user"`
	Ping       bool      `json:"ping"`
	Date       time.Time `json:"date"`
}

func ChessEventOptions(c *gin.Context) {
	c.Header("Allow", "OPTIONS,POST")
	c.Header("Access-Control-Allow-Origin", "https://www.chess.com")
	c.Header("Access-Control-Allow-Methods", "OPTIONS,POST")
	c.JSON(200, gin.H{})

}
func ChessEventCallback(c *gin.Context) {
	config := c.Value("config").(utils.Config)
	Logger := c.Value("logger").(logger.LogWrapperObj)
	body, err := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Origin", "https://www.chess.com")
	if err != nil {
		Logger.Error("unable to get Body", zap.Error(err))
		c.JSON(400, gin.H{
			"status": "ko",
		})
		return
	}

	data := &ChessEvent{}

	if err := json.Unmarshal(body, data); err != nil {
		Logger.Error("unable to decode body", zap.Error(err))
		c.JSON(400, gin.H{
			"status": "ko",
		})
		return
	}

	if validChessExtension(config, data.Channel, data.BotName, data.ApiKey) {
		c.JSON(401, gin.H{
			"status": "ko",
		})
		return
	}

	Logger.Info("Chess event", zap.Reflect("event", data))

	if data.Ping {
		c.JSON(200, gin.H{
			"status": "ok",
			"ping":   true,
		})
		return
	}

	// on va enregistrer dans la bdd le statut en cours de la partie...
	filePath := config.BddPath + "/events/" + data.Channel + "/chess/messages.json"
	now := time.Now()
	data.Date = now
	d, _ := json.Marshal(data)
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
	if err := os.Truncate(filePath, 0); err != nil {
		Logger.Error("fail to open bdd", zap.Error(err))
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		Logger.Error("fail to open bdd", zap.Error(err))
	}
	f.Write(d)
	f.Close()
	m.Unlock()

	c.JSON(200, gin.H{
		"status": "ok",
	})
}

func validChessExtension(config utils.Config, channel string, bot string, apiKey string) bool {
	channelConfig, err := utils.GetChannelConfig(config, bot, channel)
	if err != nil {
		return false
	}
	if channelConfig.ExtensionApiKey != apiKey {
		return false
	}
	return true
}
