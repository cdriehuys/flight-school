package app

import "net/http"

func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	method := r.Method
	uri := r.URL.RequestURI()

	a.logger.Error("Unhandled server error", "error", err, "method", method, "uri", uri)
	a.genericError(w, http.StatusInternalServerError)
}

// genericError writes a response containing the generic text for the provided status code.
func (a *App) genericError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
