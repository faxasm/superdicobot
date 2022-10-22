package handlers

import (
	"encoding/csv"
	"github.com/gin-gonic/gin"
	"github.com/nicklaw5/helix/v2"
	funk "github.com/thoas/go-funk"
	"go.uber.org/zap"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"superdicobot/internal/bdd"
	"superdicobot/internal/logger"
	"superdicobot/internal/oauth"
	"superdicobot/utils"

	"sync"
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
			"isConfig":       true,
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
			aliasList := strings.Split(strings.ReplaceAll(alias, "\r\n", "\n"), "\n")
			aliasesOk := funk.Map(aliasList, func(item string) string { return strings.TrimSpace(item) }).([]string)
			hotConfig.CustomCmds = append(hotConfig.CustomCmds, bdd.CustomCmd{
				Aliases:  aliasesOk,
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
			aliasList := strings.Split(strings.ReplaceAll(alias, "\r\n", "\n"), "\n")
			aliasesOk := funk.Map(aliasList, func(item string) string { return strings.TrimSpace(item) }).([]string)
			hotConfig.RewardCmds = append(hotConfig.RewardCmds, bdd.RewardCmd{
				Aliases:  aliasesOk,
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
			aliasList := strings.Split(strings.ReplaceAll(alias, "\r\n", "\n"), "\n")
			aliasesOk := funk.Map(aliasList, func(item string) string { return strings.TrimSpace(item) }).([]string)
			hotConfig.LastRewardCmds = append(hotConfig.LastRewardCmds, bdd.RewardCmd{
				Aliases:  aliasesOk,
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
			aliasList := strings.Split(strings.ReplaceAll(alias, "\r\n", "\n"), "\n")
			aliasesOk := funk.Map(aliasList, func(item string) string { return strings.TrimSpace(item) }).([]string)
			hotConfig.SoldRewardCmds = append(hotConfig.SoldRewardCmds, bdd.RewardCmd{
				Aliases:  aliasesOk,
				Cmd:      cmdSoldReward[i],
				CoolDown: coolDownSoldReward[i],
				Id:       idSoldReward[i],
				User:     userRoleSoldReward[i],
				Unit:     unitValue,
			})
		}
	}

	aliasesTotalReward := form["totalRewardCmd[aliases][]"]
	cmdTotalReward := form["totalRewardCmd[cmd][]"]
	coolDownTotalReward := form["totalRewardCmd[coolDown][]"]
	idTotalReward := form["totalRewardCmd[id][]"]
	unitTotalReward := form["totalRewardCmd[unit][]"]
	userRoleTotalReward := form["totalRewardCmd[user][]"]
	hotConfig.TotalRewardCmds = make([]bdd.RewardCmd, 0)
	for i, alias := range aliasesTotalReward {
		unitValue, err := strconv.Atoi(unitTotalReward[i])
		if err != nil {
			unitValue = 1
		}
		if alias != "" && cmdTotalReward[i] != "" {
			aliasList := strings.Split(strings.ReplaceAll(alias, "\r\n", "\n"), "\n")
			aliasesOk := funk.Map(aliasList, func(item string) string { return strings.TrimSpace(item) }).([]string)
			hotConfig.TotalRewardCmds = append(hotConfig.TotalRewardCmds, bdd.RewardCmd{
				Aliases:  aliasesOk,
				Cmd:      cmdTotalReward[i],
				CoolDown: coolDownTotalReward[i],
				Id:       idTotalReward[i],
				User:     userRoleTotalReward[i],
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
			"isConfig":       true,
		},
	)
}

func Rewards(c *gin.Context) {
	//c.String(200, csrf.GetToken(c))
	channel := c.Param("channel")
	botName := c.Param("bot")
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	channelConfig := c.Value("channelConfig").(utils.ChannelConfig)
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

	channelApiClient, err := helix.NewClient(&helix.Options{
		ClientID:        config.Webserver.Oauth.ClientId,
		ClientSecret:    config.Webserver.Oauth.ClientSecret,
		AppAccessToken:  config.Webserver.Oauth.AppToken,
		UserAccessToken: channelConfig.Token,
	})
	if err != nil {
		Logger.Error("client is OUT", zap.Error(err))
		//panic("out apiClient is down" + err.Error())
	}
	resp, err := channelApiClient.GetCustomRewards(&helix.GetCustomRewardsParams{
		BroadcasterID: channelConfig.UserId,
	})
	if err != nil {
		Logger.Warn("error with api", zap.Error(err))
	}

	Logger.Info("error with api", zap.Reflect("data", resp.Data))

	c.HTML(
		http.StatusOK,
		"views/rewards.gohtml",
		gin.H{
			"user":           user,
			"config":         safeConfig,
			"currentBot":     botName,
			"currentChannel": channel,
			"hotConfig":      hotConfig,
			"rewards":        resp.Data.ChannelCustomRewards,
			"isReward":       true,
		},
	)
}

func Redeems(c *gin.Context) {
	//c.String(200, csrf.GetToken(c))
	channel := c.Param("channel")
	botName := c.Param("bot")
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	//channelConfig := c.Value("channelConfig").(utils.ChannelConfig)
	rewardId := c.Param("rewardId")
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

	filePath := config.BddPath + "/events/" + channel + "/rewards/" + rewardId + ".csv"
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

	//Logger.Info("test", zap.Reflect("records", records))
	events := bdd.MapToEventsRewardId(records)
	lastEvents := bdd.LastByRedeemId(events)
	sort.Sort(lastEvents)

	c.HTML(
		http.StatusOK,
		"views/redeems.gohtml",
		gin.H{
			"user":           user,
			"config":         safeConfig,
			"currentBot":     botName,
			"currentChannel": channel,
			"hotConfig":      hotConfig,
			"redeems":        lastEvents,
			"isReward":       true,
		},
	)
}

func ApiKeys(c *gin.Context) {
	//c.String(200, csrf.GetToken(c))
	channel := c.Param("channel")
	botName := c.Param("bot")
	user := c.Value("user").(string)
	config := c.Value("config").(utils.Config)
	channelConfig := c.Value("channelConfig").(utils.ChannelConfig)
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
		"views/apikeys.gohtml",
		gin.H{
			"user":            user,
			"config":          safeConfig,
			"currentBot":      botName,
			"currentChannel":  channel,
			"extensionApiKey": channelConfig.ExtensionApiKey,
			"isApiKeys":       true,
		},
	)
}
