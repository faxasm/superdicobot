package logger

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"time"
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

func GinZap(logger LogWrapperObj, timeFormat string, utc bool, alwaysLog bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			for _, e := range c.Errors.Errors() {
				logger.Error(e)
			}
		} else {
			if alwaysLog || c.Writer.Status() >= 300 {
				logger.Info(path,
					zap.Int("status", c.Writer.Status()),
					zap.String("method", c.Request.Method),
					zap.String("path", path),
					zap.String("query", query),
					zap.String("ip", c.ClientIP()),
					zap.String("user-agent", c.Request.UserAgent()),
					zap.String("time", end.Format(timeFormat)),
					zap.Duration("latency", latency),
				)
			}
		}
	}
}
