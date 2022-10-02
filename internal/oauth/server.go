package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
	"io"
	"net/url"
	"strings"
	userpool "superdicobot/internal"
	"superdicobot/internal/logger"
	"time"

	"net/http"
	"superdicobot/utils"
)

const (
	stateCallbackKey = "oauth-state-callback"
	oauthSessionName = "oauth-oidc-session"
	oauthTokenKey    = "oauth-token"
)

type OState struct {
	Key      string `json:"key"`
	Redirect string `json:"redirect"`
}

var (
// oidcIssuer = "https://id.twitch.tv/oauth2"
// claims     = oauth2.SetAuthURLParam("claims", `{"id_token":{}}`)
)

func Root(c *gin.Context) {

	body := `
			<html><body><a href="/login">Login using Twitch</a></body></html>
    `
	c.Status(http.StatusOK)
	_, err := c.Writer.Write([]byte(body))
	if err != nil {
		//err
	}

}

func Login(c *gin.Context) {
	oauth2Config := c.Value("oauth2Config").(*oauth2.Config)
	session := sessions.Default(c)

	var tokenBytes [255]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "Couldn't generate a session!",
			"err": err.Error(),
		})
		return
	}

	callback, _ := c.GetQuery("callback")

	state := hex.EncodeToString(tokenBytes[:])

	oauth2State := OState{
		Key:      state,
		Redirect: callback,
	}

	jsonState, _ := json.Marshal(oauth2State)

	globalState := base64.URLEncoding.EncodeToString(jsonState)

	session.AddFlash(globalState, stateCallbackKey)

	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "Couldn't generate a session!",
			"err": err.Error(),
		})
		return
	}
	http.Redirect(c.Writer, c.Request, oauth2Config.AuthCodeURL(globalState), http.StatusTemporaryRedirect)
}

func Redirect(c *gin.Context) {
	oauth2Config := c.Value("oauth2Config").(*oauth2.Config)
	Logger := c.Value("logger").(logger.LogWrapperObj)
	session := sessions.Default(c)
	var err error

	// ensure we flush the csrf challenge even if the request is ultimately unsuccessful
	defer func() {
		if err := session.Save(); err != nil {
			Logger.Error("error saving session", zap.Error(err))
		}
	}()
	switch stateChallenge, state := session.Flashes(stateCallbackKey), c.Request.FormValue("state"); {
	case state == "", len(stateChallenge) < 1:
		err = errors.New("missing state challenge")
	case state != stateChallenge[0]:
		err = fmt.Errorf("invalid oauth state, expected '%s', got '%s'\n", state, stateChallenge[0])
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "Couldn't verify your confirmation, please try again.",
			"err": err.Error(),
		})
		return
	}

	token, err := oauth2Config.Exchange(context.Background(), c.Request.FormValue("code"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "Couldn't verify your code.",
			"err": err.Error(),
		})
		return
	}

	err = SaveTokenToSession(c, session, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "Couldn't get valid state!",
			"err": err.Error(),
		})
		return
	}
	c.Request.FormValue("state")
	decodedState, _ := base64.URLEncoding.DecodeString(c.Request.FormValue("state"))
	globalState := &OState{}
	err = json.Unmarshal(decodedState, globalState)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "Couldn't get valid state!",
			"err": err.Error(),
		})
		return
	}

	if globalState.Redirect == "/" || globalState.Redirect == "" {
		globalState.Redirect = "/admin/"
	}
	http.Redirect(c.Writer, c.Request, globalState.Redirect, http.StatusTemporaryRedirect)

}

func SaveTokenToSession(c *gin.Context, session sessions.Session, token *oauth2.Token) (err error) {

	// add the oauth token to session
	session.Options(sessions.Options{
		Path: "/",
	})
	session.Set(oauthTokenKey, *token)
	err = session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "Couldn't generate a session!",
			"err": err.Error(),
		})

		return
	}
	return
}

func CheckSession() func(ctx *gin.Context) {
	return func(c *gin.Context) {
		//oidcVerifier := c.Value("oidcVerifier").(*oidc.IDTokenVerifier)
		oauth2Config := c.Value("oauth2Config").(*oauth2.Config)
		oauthPool := c.Value("oauthPool").(*userpool.TTLOauthMap)
		Logger := c.Value("logger").(logger.LogWrapperObj)
		session := sessions.Default(c)

		q := url.Values{}
		q.Set("callback", c.FullPath())
		login := url.URL{Path: "/login", RawQuery: q.Encode()}

		var tokenKey *oauth2.Token
		if session.Get(oauthTokenKey) != nil {
			tokenData := session.Get(oauthTokenKey).(oauth2.Token)
			tokenKey = &tokenData
		} else {
			c.Redirect(http.StatusTemporaryRedirect, login.RequestURI())
			c.Abort()
			return
		}
		if tokenKey.Expiry.Before(time.Now()) {
			ts := oauth2Config.TokenSource(context.Background(), &oauth2.Token{RefreshToken: tokenKey.RefreshToken})
			tok, err := ts.Token()
			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, login.RequestURI())
				c.Abort()
				return
			}
			// update session !
			if tok.AccessToken != tokenKey.AccessToken {
				err = SaveTokenToSession(c, session, tok)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"msg": "Couldn't get valid state!",
						"err": err.Error(),
					})
					c.Abort()
					return
				}
			}
		}
		tokenData := session.Get(oauthTokenKey).(oauth2.Token)

		user := oauthPool.Get(tokenData.AccessToken)
		if user == "" {
			ts := oauth2Config.TokenSource(context.Background(), &tokenData)
			client := oauth2.NewClient(context.Background(), ts)
			response, err := client.Get("https://id.twitch.tv/oauth2/userinfo")
			if err != nil {
				Logger.Error("unable to decode msg", zap.Error(err))
				c.Abort()
				return
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"msg": "Couldn't generate a session!",
					"err": err.Error(),
				})
				c.Abort()
				return
			}
			var claim struct {
				Iss               string `json:"iss"`
				Sub               string `json:"sub"`
				Aud               string `json:"aud"`
				Exp               int32  `json:"exp"`
				Iat               int32  `json:"iat"`
				Nonce             string `json:"nonce"`
				Email             string `json:"email"`
				PreferredUsername string `json:"preferred_username"`
			}
			err = json.Unmarshal(body, &claim)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"msg": "Couldn't get user infos!",
					"err": err.Error(),
				})
				c.Abort()
				return
			}
			oauthPool.Put(tokenData.AccessToken, claim.PreferredUsername, int64(claim.Exp))
			user = oauthPool.Get(tokenData.AccessToken)
		}

		Logger.Info("display pool oauth", zap.String("pool", oauthPool.Display()))
		c.Set("user", strings.ToLower(user))
		c.Next()
	}
}

func ConfigureOauth2(config utils.Webserver) func(ctx *gin.Context) {

	oauth2Config := &oauth2.Config{
		ClientID:     config.Oauth.ClientId,
		ClientSecret: config.Oauth.ClientSecret,
		Scopes:       []string{},
		Endpoint:     twitch.Endpoint,
		RedirectURL:  config.Oauth.RedirectURL,
	}
	oauthPool := userpool.NewOauthMap(0)
	return func(c *gin.Context) {
		c.Set("oauth2Config", oauth2Config)
		c.Set("oauthPool", oauthPool)
		//c.Set("oidcVerifier", oidcVerifier)
		c.Next()
	}

}

func SecureRoute(user string, channel string, admin string) bool {
	return user == channel || user == admin
}
