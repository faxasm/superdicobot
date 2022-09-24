package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

type LogWrapperObj struct {
	Logger *zap.SugaredLogger
}

func NewLogger(level string, file string) LogWrapperObj {
	return LogWrapperObj{
		Logger: initLogger(level, file),
	}
}

func initLogger(level string, filePath string) *zap.SugaredLogger {

	zapLevel := new(zap.AtomicLevel)
	err := zapLevel.UnmarshalText([]byte(level))
	if err != nil {
		panic(err)
	}

	callerSkip := zap.AddCallerSkip(1)
	config := zap.NewProductionConfig()
	config.Level.SetLevel(zapLevel.Level())
	config.EncoderConfig.TimeKey = "date_time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
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
		config.OutputPaths = []string{"stderr", filePath}
	}

	logger, _ := config.Build()

	defer logger.Sync() // flushes buffer, if any
	return logger.With(zap.Namespace("app")).WithOptions(callerSkip).Sugar()
}

func (logWrapper LogWrapperObj) Debug(message string, fields ...zap.Field) {
	logWrapper.Logger.Desugar().Debug(message, fields...)
}

func (logWrapper LogWrapperObj) Info(message string, fields ...zap.Field) {
	logWrapper.Logger.Desugar().Info(message, fields...)
}

func (logWrapper LogWrapperObj) Warn(message string, fields ...zap.Field) {
	logWrapper.Logger.Desugar().Warn(message, fields...)
}

func (logWrapper LogWrapperObj) Error(message string, fields ...zap.Field) {
	logWrapper.Logger.Desugar().Error(message, fields...)
}

func (logWrapper LogWrapperObj) Fatal(message string, fields ...zap.Field) {
	logWrapper.Logger.Desugar().Fatal(message, fields...)
}
