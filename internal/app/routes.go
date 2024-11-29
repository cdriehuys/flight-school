package app

import (
	"net/http"

	"github.com/justinas/alice"
)

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.homepage)

	middleware := alice.New(a.logRequest)

	return middleware.Then(mux)
}
