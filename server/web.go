package server

import (
	"encoding/gob"
	"github.com/gin-gonic/autotls"
	"html/template"
	"strings"
	"superdicobot/internal/handlers"
	"superdicobot/internal/logger"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"golang.org/x/oauth2"
	"net/http"
	"superdicobot/internal/oauth"
	"superdicobot/utils"
)
import "github.com/gin-gonic/gin"

func LaunchServer(notify chan string, config utils.Config, Logger logger.LogWrapperObj) {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(logger.GinZap(Logger, time.RFC3339, true, false))

	//register statics
	r.Static("/css", "./server/web/css")
	r.Static("/img", "./server/web/img")
	r.Static("/scss", "./server/web/scss")
	r.Static("/vendor", "./server/web/vendor")
	r.Static("/js", "./server/web/js")
	r.Static("/favicon.ico", "./server/web/favicon.ico")

	r.SetFuncMap(template.FuncMap{
		"StringsJoin": strings.Join,
		"LocalDate": func(t time.Time) string {
			l, _ := time.LoadLocation("Europe/Paris")
			return t.In(l).Format("02/01/2006 15:04:05")
		},
	})
	r.LoadHTMLGlob("server/templates/**/*")

	gob.Register(oauth2.Token{})

	r.Use(utils.ConfigMiddleware(config, Logger))

	store := cookie.NewStore([]byte(config.Webserver.Oauth.CookieSecret))
	store.Options(sessions.Options{MaxAge: 365 * 60 * 60 * 24}) // expire in a day
	r.Use(sessions.Sessions("mySession", store))
	r.Use(oauth.ConfigureOauth2(config.Webserver))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", oauth.Root)
	r.GET("/login", oauth.Login)
	r.GET("/redirect", oauth.Redirect)
	r.POST("/events", handlers.EventCallback)
	r.OPTIONS("/chess/events", handlers.ChessEventOptions)
	r.POST("/chess/events", handlers.ChessEventCallback)

	restricted := r.Group("/admin")
	//register statics

	restricted.Use(oauth.CheckSession())
	restricted.GET("/", handlers.Root)

	restricted.GET("/:bot/:channel", handlers.Channel)
	restricted.POST("/:bot/:channel", handlers.PostChannel)

	restricted.GET("/:bot/:channel/rewards", handlers.Rewards)
	restricted.GET("/:bot/:channel/rewards/:rewardId", handlers.Redeems)

	restricted.GET("/:bot/:channel/apikeys", handlers.ApiKeys)

	var err error
	if len(config.Webserver.Hosts) > 0 && config.Webserver.Hosts[0] != "localhost:8080" {
		err = autotls.Run(r, config.Webserver.Hosts...)
	} else {
		err = r.Run(":8080")
	}

	if err != nil {
		notify <- "erreur server: " + err.Error()
	}
}
