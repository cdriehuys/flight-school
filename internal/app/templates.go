package app

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"

	"github.com/cdriehuys/flight-school/internal/models"
)

type templateData struct {
	AreaOfOperation  models.AreaOfOperation
	AreasOfOperation []models.AreaOfOperation
	Task             models.Task
	TaskConfidence   models.TaskConfidence
	Tasks            []models.Task
}

// render executes a template and writes it as the response.
func (app *App) render(w http.ResponseWriter, r *http.Request, status int, page string, data templateData) {
	// Render to a buffer first so that we can write a proper error message if the rendering fails.
	// If we wrote straight to the response, errors would result in a half-written page.
	buf := new(bytes.Buffer)
	err := app.templates.Render(buf, page, data)
	if err != nil {
		app.logger.Error("Failed to render template.", "error", err, "page", page)
		app.serverError(w, r, err)
		return
	}

	// If the template is written to the buffer without any errors, we are safe
	// to go ahead and write the HTTP status code to http.ResponseWriter.
	w.WriteHeader(status)

	// Write the contents of the buffer to the http.ResponseWriter. Note: this
	// is another time where we pass our http.ResponseWriter to a function that
	// takes an io.Writer.
	buf.WriteTo(w)
}

type templateEngine interface {
	Render(w io.Writer, template string, data templateData) error
}

type templateCache struct {
	logger    *slog.Logger
	templates map[string]*template.Template
}

func newTemplateCache(logger *slog.Logger, files fs.FS, funcs template.FuncMap) (templateCache, error) {
	pages, err := fs.Glob(files, "pages/*.html.tmpl")
	if err != nil {
		return templateCache{}, fmt.Errorf("failed to gather pages: %v", err)
	}

	cache := make(map[string]*template.Template)
	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"base.html.tmpl",
			"partials/*.html.tmpl",
			page,
		}

		ts, err := template.New(name).Funcs(funcs).ParseFS(files, patterns...)
		if err != nil {
			return templateCache{}, fmt.Errorf("failed to build template for %s: %v", page, err)
		}

		logger.Info("Compiled page template", "page", page)
		cache[name] = ts
	}

	return templateCache{logger, cache}, nil
}

func (c templateCache) Render(w io.Writer, name string, data templateData) error {
	template, ok := c.templates[name]
	if !ok {
		return fmt.Errorf("cache does not contain a template named %s", name)
	}

	c.logger.Debug("Executing cached template.", "name", name)

	return template.ExecuteTemplate(w, "base", data)
}

type liveTemplateLoader struct {
	logger *slog.Logger
	files  fs.FS
	funcs  template.FuncMap
}

func (l liveTemplateLoader) Render(w io.Writer, name string, data templateData) error {
	pagePath := path.Join("pages", name)

	patterns := []string{
		"base.html.tmpl",
		"partials/*.html.tmpl",
		pagePath,
	}

	ts, err := template.New(name).Funcs(l.funcs).ParseFS(l.files, patterns...)
	if err != nil {
		return fmt.Errorf("failed to build template for %s: %v", pagePath, err)
	}

	l.logger.Info("Parsed template.", "name", name)

	return ts.ExecuteTemplate(w, "base", data)
}
