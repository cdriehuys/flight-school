package app

import (
	"io/fs"
	"net/http"
	"net/url"
)

type staticfiles interface {
	URL(name string) (string, error)

	http.Handler
}

type staticDir struct {
	http.Handler
}

func newStaticDir(files fs.FS) *staticDir {
	handler := http.FileServerFS(files)

	return &staticDir{handler}
}

func (s staticDir) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "no-store")

	s.Handler.ServeHTTP(w, r)
}

func (s staticDir) URL(name string) (string, error) {
	return url.JoinPath("/static/", name)
}
