package app

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

func (a *App) serverError(w http.ResponseWriter, _ *http.Request, err error) {
	if a.debug {
		trace := debug.Stack()
		body := fmt.Sprintf("%s\n%s", err.Error(), trace)
		http.Error(w, body, http.StatusInternalServerError)
		return
	}

	a.genericError(w, http.StatusInternalServerError)
}

// genericError writes a response containing the generic text for the provided status code.
func (a *App) genericError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
