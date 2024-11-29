package app

import (
	"fmt"
	"net/http"
)

func (a *App) homepage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Hello, World!")
}
