package cli

import (
	"io"
	"log/slog"

	"github.com/spf13/viper"
)

func createLogger(logStream io.Writer) *slog.Logger {
	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}

	return slog.New(slog.NewTextHandler(logStream, &slog.HandlerOptions{Level: logLevel}))
}
