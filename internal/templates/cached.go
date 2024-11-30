package templates

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
)

func BuildCacheFromFS(files fs.FS) (map[string]*template.Template, error) {
	pages, err := fs.Glob(files, "pages/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to gather pages: %v", err)
	}

	cache := make(map[string]*template.Template)
	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"base.html.tmpl",
			// "partials/*.html.tmpl",
			page,
		}

		ts, err := template.New(name).ParseFS(files, patterns...)
		if err != nil {
			return nil, fmt.Errorf("failed to build template for %s: %v", page, err)
		}

		cache[name] = ts
	}

	return cache, nil
}
