package logging

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(level string) (*zap.Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	return getLogger(lvl), nil
}

func getLogger(
	level zapcore.Level,
) *zap.Logger {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeName = zapcore.FullNameEncoder

	ec.EncodeName = zapcore.FullNameEncoder
	ec.EncodeTime = zapcore.RFC3339TimeEncoder
	ec.EncodeDuration = zapcore.MillisDurationEncoder
	ec.EncodeLevel = zapcore.CapitalLevelEncoder
	ec.EncodeCaller = zapcore.ShortCallerEncoder
	ec.NameKey = "logger"
	ec.MessageKey = "message"
	ec.LevelKey = "level"
	ec.TimeKey = "timestamp"
	ec.CallerKey = "caller"
	ec.StacktraceKey = "trace"
	ec.LineEnding = zapcore.DefaultLineEnding
	ec.ConsoleSeparator = " "

	stdout := zapcore.AddSync(os.Stdout)

	consoleEncoder := zapcore.NewConsoleEncoder(ec)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, level),
	)

	l := zap.New(core)

	return l
}

func parseLevel(lvl string) (zapcore.Level, error) {
	switch strings.ToLower(lvl) {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	}
	return zap.InfoLevel, fmt.Errorf("invalid log level <%v>", lvl)
}
