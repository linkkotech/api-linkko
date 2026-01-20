package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new production-ready zap logger
func NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	
	// Configure encoding
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	
	// Build logger
	logger, err := config.Build(
		zap.AddCallerSkip(1), // Skip wrapper functions in stack trace
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}
