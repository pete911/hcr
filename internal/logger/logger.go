package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(level zapcore.Level) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.DisableStacktrace = true
	config.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	config.Level.SetLevel(level)
	return config.Build()
}
