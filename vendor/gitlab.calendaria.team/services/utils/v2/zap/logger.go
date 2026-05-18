package zap

import (
	kratoszap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(console bool) *kratoszap.Logger {
	encoding := "json"
	if console {
		encoding = "console"
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevel(),
		Encoding:         encoding,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
		},
	}

	if console {
		cfg.Level.SetLevel(zapcore.DebugLevel)
	}

	zapLogger := zap.Must(cfg.Build())

	return kratoszap.NewLogger(zapLogger)
}
