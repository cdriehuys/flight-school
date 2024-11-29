package app

import (
	"net/http"
	"time"
)

// logRequest logs at the beginning and end of each request. It includes the request duration, which
// means it should be placed as high as possible in the middleware chain to ensure accurate timing.
func (a *App) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := a.logger.With("method", r.Method, "path", r.URL.Path)

		logger.InfoContext(r.Context(), "Handling request")
		start := time.Now()

		next.ServeHTTP(w, r)

		elapsed := time.Since(start)
		logger.Info("Request completed", "duration", elapsed)
	})
}
