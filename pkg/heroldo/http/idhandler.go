package http

import (
	"context"
	"log/slog"
	"net/http"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// IdMiddleware returns an HTTP handler that injects a unique request ID into
// the context of each incoming request.
func IdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, err := gonanoid.New()
		if err != nil {
			writeError(w, slog.Default(), http.StatusInternalServerError, "", "failed to generate ID for request", "failed to generate ID for request")
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "request_id", requestID))

		next.ServeHTTP(w, r)
	})
}
