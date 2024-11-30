package app

import (
	"net/http"
)

func (a *App) homepage(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, http.StatusOK, "index.html.tmpl", templateData{})
}
