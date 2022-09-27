package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"superdicobot/internal/oauth"
	"superdicobot/utils"
)

func Root(c *gin.Context) {
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	safeConfig := utils.GetSafeConfig(config, user)
	c.HTML(
		http.StatusOK,
		"views/index.gohtml",
		gin.H{
			"user":           user,
			"config":         safeConfig,
			"currentBot":     "",
			"currentChannel": "",
		},
	)
}

func Channel(c *gin.Context) {
	//c.String(200, csrf.GetToken(c))
	channel := c.Param("channel")
	botName := c.Param("bot")
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	safeConfig := utils.GetSafeConfig(config, user)
	botConfig, err := utils.GetBot(config, botName)
	if err != nil {
		c.HTML(
			http.StatusNotFound,
			"views/404.gohtml",
			gin.H{
				"user":           user,
				"config":         safeConfig,
				"currentBot":     "",
				"currentChannel": "",
			},
		)
		return
	}

	if !oauth.SecureRoute(user, channel, botConfig.Administrator) {
		c.HTML(
			http.StatusNotFound,
			"views/404.gohtml",
			gin.H{
				"user":           user,
				"config":         safeConfig,
				"currentBot":     "",
				"currentChannel": "",
			},
		)
		return
	}
	c.HTML(
		http.StatusOK,
		"views/bot.gohtml",
		gin.H{
			"user":           user,
			"config":         safeConfig,
			"currentBot":     botName,
			"currentChannel": channel,
		},
	)
}

func PostChannel(c *gin.Context) {
	channel := c.Param("channel")
	botName := c.Param("bot")
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	safeConfig := utils.GetSafeConfig(config, user)
	botConfig, err := utils.GetBot(config, botName)
	if err != nil {
		c.HTML(
			http.StatusNotFound,
			"views/404.gohtml",
			gin.H{
				"user":           user,
				"config":         safeConfig,
				"currentBot":     "",
				"currentChannel": "",
			},
		)
		return
	}

	if !oauth.SecureRoute(user, channel, botConfig.Administrator) {
		c.HTML(
			http.StatusNotFound,
			"views/404.gohtml",
			gin.H{
				"user":           user,
				"config":         safeConfig,
				"currentBot":     "",
				"currentChannel": "",
			},
		)
		return
	}
	c.HTML(
		http.StatusOK,
		"views/bot.gohtml",
		gin.H{
			"user":           user,
			"config":         safeConfig,
			"currentBot":     botName,
			"currentChannel": channel,
		},
	)
}
