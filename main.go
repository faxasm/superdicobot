package main

import (
	"go.uber.org/zap"
	"superdicobot/internal/bot"
	"superdicobot/internal/logger"
	"superdicobot/server"
	"superdicobot/utils"
)

func main() {
	config, errConfig := utils.LoadConfig("./config")
	if errConfig != nil {
		panic("errConfig: " + errConfig.Error())
	}
	Logger := logger.NewLogger(config.LoggerLevel, config.LoggerFile)
	notify := make(chan string)

	//connecting foreach channel
	for _, botConfig := range config.Bots {
		go bot.NewBot(notify, botConfig, config)
	}

	//start webserver
	go server.LaunchServer(notify, config, Logger)

	msg := <-notify
	Logger.Error("panic for at least one bot", zap.String("notify", msg))
	Logger.Info("this is the end.")
}
