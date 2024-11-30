package app

import (
	"html/template"
	"log/slog"
)

type App struct {
	logger    *slog.Logger
	templates map[string]*template.Template
}

func New(logger *slog.Logger, templates map[string]*template.Template) *App {
	return &App{
		logger:    logger,
		templates: templates,
	}
}
