package app

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/cdriehuys/flight-school/internal/models"
)

type templateData struct {
	AreaOfOperation  models.AreaOfOperation
	AreasOfOperation []models.AreaOfOperation
	Task             models.Task
	TaskConfidence   models.Confidence
	Tasks            []models.TaskSummary
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
	allFuncs := templateFuncs(funcs)

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

		ts, err := template.New(name).Funcs(allFuncs).ParseFS(files, patterns...)
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

func makeLiveTemplateLoader(logger *slog.Logger, files fs.FS, funcs template.FuncMap) liveTemplateLoader {
	return liveTemplateLoader{
		logger: logger,
		files:  files,
		funcs:  templateFuncs(funcs),
	}
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

// templateFuncs creates a function map for templates by merging the provided functions into a base
// set of functionality.
func templateFuncs(custom template.FuncMap) template.FuncMap {
	funcs := template.FuncMap{
		"add":                add,
		"confidenceButton":   confidenceButton,
		"confidenceFormData": makeConfidenceFormData,
		"fracAsPercent":      fracAsPercent,
		"join":               strings.Join,
	}

	for k, f := range custom {
		funcs[k] = f
	}

	return funcs
}

// add provides basic addition inside a template
func add(a int32, b int32) int32 {
	return a + b
}

// fracAsPercent computes an integer percentage in the range [0, 100] from a fraction. An undefined
// fraction is treated as 100%.
func fracAsPercent(numerator int, denominator int) int {
	if denominator == 0 {
		return 100
	}

	decimal := float64(numerator) / float64(denominator) * 100

	return int(math.Round(decimal))
}

type confidenceFormData struct {
	ElementID       int32
	ConfidenceLevel *models.ConfidenceLevel
}

func makeConfidenceFormData(elementID int32, level *models.ConfidenceLevel) confidenceFormData {
	return confidenceFormData{elementID, level}
}

func confidenceButton(rawLevel int32, current *models.ConfidenceLevel) template.HTML {
	level := models.ConfidenceLevel(rawLevel)
	classes := []string{"button-group__btn"}

	var icon string
	var name string

	switch level {
	case models.ConfidenceLevelLow:
		classes = append(classes, "button-group__btn--bad")
		icon = "fa-face-frown"
		name = "low"

	case models.ConfidenceLevelMedium:
		classes = append(classes, "button-group__btn--meh")
		icon = "fa-face-meh"
		name = "medium"

	case models.ConfidenceLevelHigh:
		classes = append(classes, "button-group__btn--happy")
		icon = "fa-smile"
		name = "high"
	}

	if current != nil && level == *current {
		classes = append(classes, "button-group__btn--active")
	}

	classList := strings.Join(classes, " ")

	rawHTML := `<button class="` + classList + `" name="` + name + `" type="submit">
		<i class="fa-regular ` + icon + `"></i>
		</button>`

	return template.HTML(rawHTML)
}
