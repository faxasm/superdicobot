package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"superdicobot/internal/bdd"
	"superdicobot/internal/logger"
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
	Logger := c.Value("logger").(logger.LogWrapperObj)
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
	hotConfig, err := bdd.GetBddConfig(config, botName, channel, Logger)
	c.HTML(
		http.StatusOK,
		"views/bot.gohtml",
		gin.H{
			"user":           user,
			"config":         safeConfig,
			"currentBot":     botName,
			"currentChannel": channel,
			"hotConfig":      hotConfig,
		},
	)
}

func PostChannel(c *gin.Context) {
	channel := c.Param("channel")
	botName := c.Param("bot")
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	Logger := c.Value("logger").(logger.LogWrapperObj)
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

	err = c.Request.ParseForm()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "unable to parse form",
			"err": err.Error(),
		})
		c.Abort()
		return
	}
	//activeBot := c.Request.FormValue("activate") == "on"

	form := c.Request.PostForm
	activeBot := form.Get("activate") == "on"

	hotConfig, err := bdd.GetBddConfig(config, botName, channel, Logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "internal error",
			"err": err.Error(),
		})
		c.Abort()
		return
	}

	hotConfig.Activate = activeBot

	unTimeoutCmd := form["unTimeout[cmd]"][0]
	unTimeoutMaxDuration := form["unTimeout[maxTimeout]"][0]

	unTimeoutMaxDurationValue, err := strconv.Atoi(unTimeoutMaxDuration)
	if err != nil {
		unTimeoutMaxDurationValue = 0
	}

	hotConfig.UnTimeoutCmd = bdd.UnTimeoutCmd{
		Cmd:                unTimeoutCmd,
		MaxTimeoutDuration: unTimeoutMaxDurationValue,
	}
	aliases := form["customCmd[aliases][]"]
	cmd := form["customCmd[cmd][]"]
	coolDown := form["customCmd[coolDown][]"]
	userRole := form["customCmd[user][]"]
	hotConfig.CustomCmds = make([]bdd.CustomCmd, 0)
	for i, alias := range aliases {

		if alias != "" && cmd[i] != "" {
			hotConfig.CustomCmds = append(hotConfig.CustomCmds, bdd.CustomCmd{
				Aliases:  []string{alias},
				Cmd:      cmd[i],
				CoolDown: coolDown[i],
				User:     userRole[i],
			})
		}
	}

	aliasesReward := form["rewardCmd[aliases][]"]
	cmdReward := form["rewardCmd[cmd][]"]
	coolDownReward := form["rewardCmd[coolDown][]"]
	idReward := form["rewardCmd[id][]"]
	userRoleReward := form["rewardCmd[user][]"]
	hotConfig.RewardCmds = make([]bdd.RewardCmd, 0)
	for i, alias := range aliasesReward {
		if alias != "" && cmdReward[i] != "" {
			hotConfig.RewardCmds = append(hotConfig.RewardCmds, bdd.RewardCmd{
				Aliases:  []string{aliasesReward[i]},
				Cmd:      cmdReward[i],
				CoolDown: coolDownReward[i],
				Id:       idReward[i],
				User:     userRoleReward[i],
			})
		}
	}

	aliasesLastReward := form["lastRewardCmd[aliases][]"]
	cmdLastReward := form["lastRewardCmd[cmd][]"]
	coolDownLastReward := form["lastRewardCmd[coolDown][]"]
	idLastReward := form["lastRewardCmd[id][]"]
	userRoleLastReward := form["lastRewardCmd[user][]"]
	hotConfig.LastRewardCmds = make([]bdd.RewardCmd, 0)
	for i, alias := range aliasesLastReward {
		if alias != "" && cmdLastReward[i] != "" {
			hotConfig.LastRewardCmds = append(hotConfig.LastRewardCmds, bdd.RewardCmd{
				Aliases:  []string{aliasesLastReward[i]},
				Cmd:      cmdLastReward[i],
				CoolDown: coolDownLastReward[i],
				Id:       idLastReward[i],
				User:     userRoleLastReward[i],
			})
		}
	}

	aliasesSoldReward := form["soldRewardCmd[aliases][]"]
	cmdSoldReward := form["soldRewardCmd[cmd][]"]
	coolDownSoldReward := form["soldRewardCmd[coolDown][]"]
	idSoldReward := form["soldRewardCmd[id][]"]
	unitSoldReward := form["soldRewardCmd[unit][]"]
	userRoleSoldReward := form["soldRewardCmd[user][]"]
	hotConfig.SoldRewardCmds = make([]bdd.RewardCmd, 0)
	for i, alias := range aliasesSoldReward {
		unitValue, err := strconv.Atoi(unitSoldReward[i])
		if err != nil {
			unitValue = 1
		}
		if alias != "" && cmdSoldReward[i] != "" {
			hotConfig.SoldRewardCmds = append(hotConfig.SoldRewardCmds, bdd.RewardCmd{
				Aliases:  []string{aliasesSoldReward[i]},
				Cmd:      cmdSoldReward[i],
				CoolDown: coolDownSoldReward[i],
				Id:       idSoldReward[i],
				User:     userRoleSoldReward[i],
				Unit:     unitValue,
			})
		}
	}

	idCronReward := form["cronRewardCmd[id][]"]
	periodCronReward := form["cronRewardCmd[period][]"]
	soldPositiveCronReward := form["cronRewardCmd[soldPositive][]"]
	actionPositiveCronReward := form["cronRewardCmd[actionPositive][]"]
	soldActionPositiveCronReward := form["cronRewardCmd[soldActionPositive][]"]
	untiCronReward := form["cronRewardCmd[unit][]"]

	hotConfig.CronRewardCmds = make([]bdd.CronRewardCmd, 0)
	for i, id := range idCronReward {
		periodValue, err := strconv.Atoi(periodCronReward[i])
		if err != nil {
			periodValue = 0
		}
		unitValue, err := strconv.Atoi(untiCronReward[i])
		if err != nil {
			unitValue = 1
		}

		hotConfig.CronRewardCmds = append(hotConfig.CronRewardCmds, bdd.CronRewardCmd{
			Id:                 id,
			Period:             periodValue,
			SoldPositive:       soldPositiveCronReward[i],
			ActionPositive:     actionPositiveCronReward[i],
			SoldActionPositive: soldActionPositiveCronReward[i],
			Unit:               unitValue,
		})

	}

	if err := bdd.SaveBddConfig(config, botName, channel, Logger, hotConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "internal error",
			"err": err.Error(),
		})
		c.Abort()
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
			"hotConfig":      hotConfig,
		},
	)
}
