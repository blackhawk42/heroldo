package http

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/blackhawk42/heroldo/pkg/heroldo"
)

// bearerRegex matches a Bearer token from an Authorization header.
var bearerRegex = regexp.MustCompile(`Bearer (\S+)`)

// TokenAuthMiddleware returns an HTTP handler that validates the incoming
// request has a valid Bearer token using the provided TokenRegistry.
func TokenAuthMiddleware(next http.Handler, tokenRegistry heroldo.TokenRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, ok := r.Context().Value("request_id").(string)
		if !ok {
			writeError(w, slog.Default(), http.StatusInternalServerError, "", "request context missing request_id", "request context missing request_id")
			return
		}
		logger := slog.With("request_id", requestID)

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, logger, http.StatusUnauthorized, requestID, "authorization token not provided", "authorization token not provided")
			return
		}

		tokenMatch := bearerRegex.FindStringSubmatch(authHeader)
		if len(tokenMatch) != 2 {
			writeError(w, logger, http.StatusBadRequest, requestID, "unrecognized format for authorization header", "unrecognized format for authorization header", "header_content", authHeader)
			return
		}

		token := tokenMatch[1]

		user, err := tokenRegistry.VerifyToken(token)
		if err != nil {
			writeError(w, logger, http.StatusInternalServerError, requestID, "error while authenticating token", "error while authenticating token", "err", err)
			return
		}

		if user == "" {
			writeError(w, logger, http.StatusUnauthorized, requestID, "token is not authorized", "token is not authorized")
			return
		}

		logger.Info("user authorized", "username", user)

		next.ServeHTTP(w, r)
	})
}
