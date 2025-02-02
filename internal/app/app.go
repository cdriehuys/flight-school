package app

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"

	"github.com/cdriehuys/flight-school/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	logger      *slog.Logger
	templates   templateEngine
	staticfiles staticfiles

	acsModel acsModel

	debug bool
}

type Options struct {
	// Debug causes error information to be rendered as part of the response.
	Debug bool

	// LiveTemplates loads templates on each request instead of caching them at startup.
	LiveTemplates bool
}

type acsModel interface {
	GetAreaByID(ctx context.Context, acs string, areaID string) (models.AreaOfOperation, error)
	GetTaskByArea(ctx context.Context, acs string, areaID string, taskID string) (models.Task, error)
	GetTaskByElementID(ctx context.Context, elementID int32) (models.Task, error)
	GetTaskConfidence(ctx context.Context, taskID int32) (models.Confidence, error)
	ListAreasByACS(ctx context.Context, acs string) ([]models.AreaOfOperation, error)
	ListTasksByArea(ctx context.Context, areaID int32) ([]models.TaskSummary, error)
	GetElementPublicIDByID(ctx context.Context, elementID int32) (string, error)
	SetElementConfidence(ctx context.Context, elementID int32, confidence models.ConfidenceLevel) error
	ClearElementConfidence(ctx context.Context, elementID int32) error
}

func New(
	logger *slog.Logger,
	templateFiles fs.FS,
	staticFiles fs.FS,
	db *pgxpool.Pool,
	options *Options,
) (*App, error) {
	if options == nil {
		options = &Options{}
	}

	sf := newStaticDir(staticFiles)
	funcMap := template.FuncMap{
		"static": sf.URL,
	}

	var templates templateEngine
	if options.LiveTemplates {
		templates = makeLiveTemplateLoader(logger, templateFiles, funcMap)
	} else {
		var err error
		templates, err = newTemplateCache(logger, templateFiles, funcMap)
		if err != nil {
			return nil, fmt.Errorf("failed to build app templates: %v", err)
		}
	}

	acsModel := models.NewACSModel(logger, db)

	app := &App{
		logger:      logger,
		templates:   templates,
		staticfiles: sf,
		acsModel:    acsModel,
		debug:       options.Debug,
	}

	return app, nil
}
