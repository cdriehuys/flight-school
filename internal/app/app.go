package app

import "log/slog"

type App struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *App {
	return &App{
		logger: logger,
	}
}
