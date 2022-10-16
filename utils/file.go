package utils

import (
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"superdicobot/internal/logger"
	"sync"
)

func GetFile(filePath string, Logger logger.LogWrapperObj) (bool, *os.File) {
	var m sync.Mutex
	m.Lock()
	defer m.Unlock()
	f, err := os.Open(filePath)
	if err != nil {
		Logger.Error("Unable to read input file "+filePath, zap.Error(err))
		if err := f.Close(); err != nil {
			Logger.Error("Unable to close input file "+filePath, zap.Error(err))
		}
		return false, nil
	}
	if err := f.Close(); err != nil {
		Logger.Error("Unable to close input file "+filePath, zap.Error(err))
		return false, nil
	}
	return true, f
}

func GetFileContent(filePath string, Logger logger.LogWrapperObj) (bool, []byte) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		Logger.Info("Unable to read input file "+filePath, zap.Error(err))
		return false, content
	}
	return true, content
}

func SaveFileContent(filePath string, data []byte, Logger logger.LogWrapperObj) {
	var m sync.Mutex
	m.Lock()
	defer m.Unlock()
	if filePath != "" {
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				panic(err)
			}
			_, err := os.Create(filePath)
			if err != nil {
				Logger.Error("fail to create dire for Config", zap.Error(err))
				return

			}
		}
	}
	err := os.WriteFile(filePath, data, os.ModePerm)
	if err != nil {
		Logger.Info("Unable to create  file "+filePath, zap.Error(err))
		return
	}
	return
}
