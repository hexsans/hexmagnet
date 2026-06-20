// Package httpserver provides log handling using zap package.
// Code structure based on ginrus package.
package httpserver

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type GinzapFn func(c *gin.Context) []zapcore.Field

type ZapLogger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type GinzapConfig struct {
	TimeFormat string
	UTC        bool
	SkipPaths  []string
	Context    GinzapFn
}

func Ginzap(logger ZapLogger, timeFormat string, utc bool) gin.HandlerFunc {
	return WithGinzapConfig(logger, &GinzapConfig{TimeFormat: timeFormat, UTC: utc})
}

func WithGinzapConfig(logger ZapLogger, conf *GinzapConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool, len(conf.SkipPaths))
	for _, path := range conf.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		if _, ok := skipPaths[path]; !ok {
			end := time.Now()
			latency := end.Sub(start)

			if conf.UTC {
				end = end.UTC()
			}

			fields := []zapcore.Field{
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.Duration("latency", latency),
			}
			if conf.TimeFormat != "" {
				fields = append(fields, zap.String("time", end.Format(conf.TimeFormat)))
			}

			if conf.Context != nil {
				fields = append(fields, conf.Context(c)...)
			}

			if len(c.Errors) > 0 {
				for _, e := range c.Errors.Errors() {
					logger.Error(e, fields...)
				}
			} else {
				logger.Debug(path, fields...)
			}
		}
	}
}
