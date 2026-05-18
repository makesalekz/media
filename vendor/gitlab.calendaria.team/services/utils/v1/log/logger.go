package log

import (
	"os"

	kratoszap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewStdLogger() *kratoszap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevel(),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
		},
	}

	if os.Getenv("DEBUG") != "" {
		cfg.Level.SetLevel(zapcore.DebugLevel)
	}

	zapLogger := zap.Must(cfg.Build())

	klogger := kratoszap.NewLogger(zapLogger)

	return klogger
}
