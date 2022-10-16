package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/resty.v1"
	"strings"
	"superdicobot/internal/logger"
	"superdicobot/utils"
)

type ChessClient struct {
	Config utils.Config
	Logger logger.LogWrapperObj
}

type ChessStats struct {
	ChessDaily struct {
		Last struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
			Rd     int `json:"rd"`
		} `json:"last"`
		Best struct {
			Rating int    `json:"rating"`
			Date   int    `json:"date"`
			Game   string `json:"game"`
		} `json:"best"`
		Record struct {
			Win            int `json:"win"`
			Loss           int `json:"loss"`
			Draw           int `json:"draw"`
			TimePerMove    int `json:"time_per_move"`
			TimeoutPercent int `json:"timeout_percent"`
		} `json:"record"`
	} `json:"chess_daily"`
	ChessRapid struct {
		Last struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
			Rd     int `json:"rd"`
		} `json:"last"`
		Best struct {
			Rating int    `json:"rating"`
			Date   int    `json:"date"`
			Game   string `json:"game"`
		} `json:"best"`
		Record struct {
			Win  int `json:"win"`
			Loss int `json:"loss"`
			Draw int `json:"draw"`
		} `json:"record"`
	} `json:"chess_rapid"`
	ChessBullet struct {
		Last struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
			Rd     int `json:"rd"`
		} `json:"last"`
		Best struct {
			Rating int    `json:"rating"`
			Date   int    `json:"date"`
			Game   string `json:"game"`
		} `json:"best"`
		Record struct {
			Win  int `json:"win"`
			Loss int `json:"loss"`
			Draw int `json:"draw"`
		} `json:"record"`
	} `json:"chess_bullet"`
	ChessBlitz struct {
		Last struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
			Rd     int `json:"rd"`
		} `json:"last"`
		Best struct {
			Rating int    `json:"rating"`
			Date   int    `json:"date"`
			Game   string `json:"game"`
		} `json:"best"`
		Record struct {
			Win  int `json:"win"`
			Loss int `json:"loss"`
			Draw int `json:"draw"`
		} `json:"record"`
	} `json:"chess_blitz"`
	Fide    int `json:"fide"`
	Tactics struct {
		Highest struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
		} `json:"highest"`
		Lowest struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
		} `json:"lowest"`
	} `json:"tactics"`
	Lessons struct {
		Highest struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
		} `json:"highest"`
		Lowest struct {
			Rating int `json:"rating"`
			Date   int `json:"date"`
		} `json:"lowest"`
	} `json:"lessons"`
	PuzzleRush struct {
		Best struct {
			TotalAttempts int `json:"total_attempts"`
			Score         int `json:"score"`
		} `json:"best"`
	} `json:"puzzle_rush"`
}

type ChessUserArchives struct {
	Archives []string `json:"archives"`
}

type ChessVsResult struct {
	Win   int
	Loose int
	Draw  int
}

type ChessUserMonthlyArchive struct {
	Games []struct {
		URL          string `json:"url"`
		Pgn          string `json:"pgn"`
		TimeControl  string `json:"time_control"`
		EndTime      int    `json:"end_time"`
		Rated        bool   `json:"rated"`
		Tcn          string `json:"tcn"`
		UUID         string `json:"uuid"`
		InitialSetup string `json:"initial_setup"`
		Fen          string `json:"fen"`
		StartTime    int    `json:"start_time"`
		TimeClass    string `json:"time_class"`
		Rules        string `json:"rules"`
		White        struct {
			Rating   int    `json:"rating"`
			Result   string `json:"result"`
			ID       string `json:"@id"`
			Username string `json:"username"`
			UUID     string `json:"uuid"`
		} `json:"white"`
		Black struct {
			Rating   int    `json:"rating"`
			Result   string `json:"result"`
			ID       string `json:"@id"`
			Username string `json:"username"`
			UUID     string `json:"uuid"`
		} `json:"black"`
		Tournament string `json:"tournament,omitempty"`
		Match      string `json:"match,omitempty"`
	} `json:"games"`
}

const ChessApi = "https://api.chess.com"
const StatsUri = "/pub/player/%s/stats"
const ArchiveUri = "/pub/player/%s/games/archives"

func (chessClient *ChessClient) GetStats(login string) (bool, *ChessStats) {
	client := resty.New()
	uri := fmt.Sprintf(StatsUri, login)
	stats := &ChessStats{}

	r, err := client.R().Get(fmt.Sprintf("%s%s", ChessApi, uri))
	if err != nil {
		chessClient.Logger.Warn("unable to get Stats", zap.Error(err))
		return false, stats
	}

	if r.IsError() {
		chessClient.Logger.Info("unable to get Stats", zap.Int("status", r.StatusCode()), zap.ByteString("response", r.Body()))
		return false, stats
	}

	if err = json.Unmarshal(r.Body(), stats); err != nil {
		chessClient.Logger.Warn("unable to get EloMax", zap.Error(err))
		return false, stats
	}
	return true, stats
}

func (chessClient *ChessClient) ChessVsWithCache(loginCached string, loginVs string) (string, *ChessVsResult) {

	loginVs = strings.ToLower(loginVs)
	loginCached = strings.ToLower(loginCached)
	uriLoginCached := fmt.Sprintf(ArchiveUri, loginCached)
	results := &ChessVsResult{}

	archiveLoginCachedResponse, err := resty.New().R().Get(fmt.Sprintf("%s%s", ChessApi, uriLoginCached))
	if err != nil {
		chessClient.Logger.Warn("unable to get user archive", zap.Error(err))
		return loginCached, results
	}
	if archiveLoginCachedResponse.IsError() {
		chessClient.Logger.Info("error when fetchuser archive", zap.Int("status", archiveLoginCachedResponse.StatusCode()), zap.ByteString("response", archiveLoginCachedResponse.Body()))
		return loginCached, results
	}

	uriLoginVs := fmt.Sprintf(ArchiveUri, loginVs)

	archiveLoginVsResponse, err := resty.New().R().Get(fmt.Sprintf("%s%s", ChessApi, uriLoginVs))
	if err != nil {
		chessClient.Logger.Warn("unable to get user archive", zap.Error(err))
		return loginVs, results
	}
	if archiveLoginVsResponse.IsError() {
		chessClient.Logger.Info("unable to get user archive", zap.Int("status", archiveLoginVsResponse.StatusCode()), zap.ByteString("response", archiveLoginVsResponse.Body()))
		return loginVs, results
	}

	archiveLoginCached := &ChessUserArchives{}
	if err = json.Unmarshal(archiveLoginCachedResponse.Body(), archiveLoginCached); err != nil {
		return loginCached, results
	}

	for _, archiveMonthly := range archiveLoginCached.Archives {
		if isOk, chessUserMonthlyArchive := chessClient.getOrCreateArchiveInCache(archiveMonthly); isOk {
			chessClient.addVsStat(loginVs, chessUserMonthlyArchive, results)
		}
	}

	return "", results
}

func (chessClient *ChessClient) addVsStat(userVs string, archive *ChessUserMonthlyArchive, result *ChessVsResult) {
	for _, game := range archive.Games {
		if strings.ToLower(game.Black.Username) == userVs || strings.ToLower(game.White.Username) == userVs {
			chessClient.Logger.Info("found match", zap.Reflect("match", game))
			switch game.Black.Result {
			case "win":
				if userVs == strings.ToLower(game.Black.Username) {
					result.Loose += 1
				} else {
					result.Win += 1
				}
			case "agreed", "repetition", "stalemate", "insufficient", "50move", "timevsinsufficient":
				result.Draw += 1
			case "lose", "timeout", "resigned", "checkmated":
				if userVs == strings.ToLower(game.Black.Username) {
					result.Win += 1
				} else {
					result.Loose += 1
				}
			default:
			}
		}
	}
}

func (chessClient *ChessClient) getOrCreateArchiveInCache(archive string) (bool, *ChessUserMonthlyArchive) {
	cacheFile := base64.StdEncoding.EncodeToString([]byte(archive))
	chessUserMonthlyArchive := &ChessUserMonthlyArchive{}
	filePath := chessClient.Config.CachePath + "/chess/" + cacheFile
	hasContent, fileContent := utils.GetFileContent(filePath, chessClient.Logger)
	if !hasContent {
		archiveResponse, err := resty.New().R().Get(archive)
		if err != nil {
			chessClient.Logger.Warn("unable to getUserArchives", zap.Error(err))
			return false, chessUserMonthlyArchive
		}
		data := archiveResponse.Body()
		if err = json.Unmarshal(data, chessUserMonthlyArchive); err != nil {
			return false, chessUserMonthlyArchive
		}
		// chess data is ok saveIntoFile
		utils.SaveFileContent(filePath, data, chessClient.Logger)
		return true, chessUserMonthlyArchive
	}
	if err := json.Unmarshal(fileContent, chessUserMonthlyArchive); err != nil {
		return false, chessUserMonthlyArchive
	}
	return true, chessUserMonthlyArchive
}
