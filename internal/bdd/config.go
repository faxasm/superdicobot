package bdd

import (
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"superdicobot/internal/logger"
	"superdicobot/utils"
	"sync"
)

type BotHotConfig struct {
	Activate       bool            `yaml:"activate"`
	UnTimeoutCmd   UnTimeoutCmd    `yaml:"unTimeoutCmd"`
	CustomCmds     []CustomCmd     `yaml:"customCmds"`
	RewardCmds     []RewardCmd     `yaml:"rewardCmds"`
	LastRewardCmds []RewardCmd     `yaml:"lastRewardCmds"`
	SoldRewardCmds []RewardCmd     `yaml:"soldRewardCmds"`
	CronRewardCmds []CronRewardCmd `yaml:"cronRewardCmds"`
}

type UnTimeoutCmd struct {
	Cmd                string `yaml:"Cmd"`
	MaxTimeoutDuration int    `yaml:"maxTimeoutDuration"`
}

type CronRewardCmd struct {
	Id                 string `yaml:"id"`
	Period             int    `yaml:"period"`
	SoldPositive       string `yaml:"soldPositive"`
	ActionPositive     string `yaml:"actionPositive"`
	SoldActionPositive string `yaml:"SoldActionPositive"`
	Unit               int    `yaml:"unit"`
}
type CustomCmd struct {
	Aliases  []string `yaml:"aliases"`
	Cmd      string   `yaml:"cmd"`
	CoolDown string   `yaml:"coolDown"`
	User     string   `yaml:"user"`
}
type RewardCmd struct {
	Aliases  []string `yaml:"aliases"`
	Cmd      string   `yaml:"cmd"`
	CoolDown string   `yaml:"coolDown"`
	Id       string   `yaml:"id"`
	User     string   `yaml:"user"`
	Unit     int      `yaml:"unit"`
}

func GetBddConfig(config utils.Config, bot string, channel string, Logger logger.LogWrapperObj) (*BotHotConfig, error) {
	filePath := config.BddPath + "config/" + bot + "/" + channel + ".yaml"
	botHotConfig := &BotHotConfig{}
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
				Logger.Error("fail to create dire for Config", zap.Error(err))
				return nil, err
			}
		}
	}

	f, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		Logger.Error("fail to open bdd get Config", zap.Error(err))
		return nil, err
	}
	if err := f.Close(); err != nil {
		Logger.Error("fail to close bdd", zap.Error(err))
		return nil, err
	}
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		Logger.Error("fail to open yaml file", zap.Error(err))
		return nil, err
	}
	m.Unlock()
	err = yaml.Unmarshal(yamlFile, botHotConfig)
	if err != nil {
		Logger.Error("fail to unmarshall yaml", zap.Error(err))
		return nil, err
	}
	return botHotConfig, nil
}

func SaveBddConfig(config utils.Config, bot string, channel string, Logger logger.LogWrapperObj, botHotConfig *BotHotConfig) error {
	filePath := config.BddPath + "config/" + bot + "/" + channel + ".yaml"
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
				Logger.Error("fail to create dire for Config", zap.Error(err))
				return nil
			}
		}
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		Logger.Error("fail to open bdd", zap.Error(err))
		return err
	}
	data, _ := yaml.Marshal(botHotConfig)
	if _, err := f.Write(data); err != nil {
		Logger.Error("fail to write on bdd", zap.Error(err))
		return err
	}
	if err := f.Close(); err != nil {
		Logger.Error("fail to close bdd", zap.Error(err))
		return err
	}
	m.Unlock()
	return nil
}
