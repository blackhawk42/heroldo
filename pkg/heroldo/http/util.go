package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// SuccessResponse is the JSON envelope returned when a request is
// successfully accepted (HTTP 202 Accepted).
type SuccessResponse struct {
	ResponseCode int      `json:"response_code"`
	RequestID    string   `json:"request_id"`
	Channels     []string `json:"channels"`
}

// ErrorResponse is the JSON envelope returned when a request fails
// validation or processing.
type ErrorResponse struct {
	ResponseCode int    `json:"response_code"`
	RequestID    string `json:"request_id"`
	Error        string `json:"error"`
}

// writeError writes an error JSON response with the given status code, logs
// the message at Error level, and returns. The caller should return afterwards.
func writeError(w http.ResponseWriter, logger *slog.Logger, status int, requestID, userMsg, logMsg string, logArgs ...any) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{ResponseCode: status, RequestID: requestID, Error: userMsg})
	logger.Error(logMsg, logArgs...)
}

// writeSuccess writes a success JSON response with the given status code, logs
// the message at Info level, and returns. The caller should return afterwards.
func writeSuccess(w http.ResponseWriter, logger *slog.Logger, status int, requestID string, channels []string, logMsg string, logArgs ...any) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(SuccessResponse{ResponseCode: status, RequestID: requestID, Channels: channels})
	logger.Info(logMsg, logArgs...)
}
