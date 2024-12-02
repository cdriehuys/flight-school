package app

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
)

type App struct {
	logger      *slog.Logger
	templates   templateEngine
	staticfiles staticfiles

	debug bool
}

type Options struct {
	// Debug causes error information to be rendered as part of the response.
	Debug bool

	// LiveTemplates loads templates on each request instead of caching them at startup.
	LiveTemplates bool
}

func New(logger *slog.Logger, templateFiles fs.FS, staticFiles fs.FS, options *Options) (*App, error) {
	if options == nil {
		options = &Options{}
	}

	var sf staticfiles
	if options.LiveTemplates {
		sf = newStaticDir(staticFiles)
	} else {
		panic("We should have implemented hashed static files (but we didn't)")
	}

	funcMap := template.FuncMap{
		"static": sf.URL,
	}

	var templates templateEngine
	if options.LiveTemplates {
		templates = liveTemplateLoader{logger, templateFiles, funcMap}
	} else {
		var err error
		templates, err = newTemplateCache(logger, templateFiles, funcMap)
		if err != nil {
			return nil, fmt.Errorf("failed to build app templates: %v", err)
		}
	}

	app := &App{
		logger:      logger,
		templates:   templates,
		staticfiles: sf,
		debug:       options.Debug,
	}

	return app, nil
}
