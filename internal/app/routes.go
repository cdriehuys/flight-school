package app

import (
	"net/http"

	"github.com/justinas/alice"
)

func (a *App) Routes() http.Handler {
	homepageRedirect := http.RedirectHandler("/", http.StatusTemporaryRedirect)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", a.staticfiles))

	mux.HandleFunc("GET /{$}", a.homepage)
	mux.Handle("GET /acs", homepageRedirect)
	mux.Handle("GET /acs/{acs}", homepageRedirect)
	mux.HandleFunc("GET /acs/{acs}/{areaID}", a.areaDetail)

	middleware := alice.New(a.logRequest)

	return middleware.Then(mux)
}
