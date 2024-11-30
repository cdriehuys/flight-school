package app

import (
	"fmt"
	"io/fs"
	"log/slog"
)

type App struct {
	logger    *slog.Logger
	templates templateEngine

	debug bool
}

type Options struct {
	// Debug causes error information to be rendered as part of the response.
	Debug bool

	// LiveTemplates loads templates on each request instead of caching them at startup.
	LiveTemplates bool
}

func New(logger *slog.Logger, templateFiles fs.FS, options *Options) (*App, error) {
	if options == nil {
		options = &Options{}
	}

	var templates templateEngine
	if options.LiveTemplates {
		templates = liveTemplateLoader{logger, templateFiles}
	} else {
		var err error
		templates, err = newTemplateCache(logger, templateFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to build app templates: %v", err)
		}
	}

	app := &App{
		logger:    logger,
		templates: templates,
		debug:     options.Debug,
	}

	return app, nil
}
