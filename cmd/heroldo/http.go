// This file implements the HTTP handler that accepts multipart form uploads,
// validates them, and passes requests to the Discord sender.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/r/blackhawk42/heroldo/pkg/heroldo"
	"github.com/r/blackhawk42/heroldo/pkg/set"
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

// trueFalseParse is a simple helper to parse "true" and "false" type strings.
//
// Current behaviour: returns true when s is the string "true" and false with everything else.
func trueFalseParse(s string) bool {
	if s == "true" {
		return true
	} else {
		return false
	}
}

// joinTexts is a helper to join multiple "text" fieldsin the multipart.
//
// Current behaviour: simply join them all with newline separators
func joinTexts(txts []string) string {
	return strings.Join(txts, "\n")
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

// RequestHandler returns an http.Handler that parses a multipart form request.
// validates file/spoiler/content-type counts, and relays the content to the
// Discord sender.
//
// It returns JSON responses with appropriate HTTP status codes.
func RequestHandler(maxBodySize int64, sender *DiscordSender) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		request := heroldo.Request{}

		var err error
		request.RequestID, err = gonanoid.New()
		if err != nil {
			writeError(w, slog.Default(), http.StatusInternalServerError, "", "failed to generate ID for request", "failed to generate ID for request")
			return
		}

		logger := slog.With("request_id", request.RequestID)

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		err = r.ParseMultipartForm(maxBodySize)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				writeError(w, logger, http.StatusRequestEntityTooLarge, request.RequestID, "request body too large", "request body too large", "error", err)
				return
			}

			writeError(w, logger, http.StatusBadRequest, request.RequestID, fmt.Sprintf("request failed to be parsed as a multiform request: %s", err), "request failed to be parsed as a multiform request", "error", err)
			return
		}
		logger.Debug("multiform parsed")
		defer r.MultipartForm.RemoveAll()

		// Empty text and spoilers are acceptable, so no ok checking.
		text := joinTexts(r.MultipartForm.Value["text"])
		logger.Debug("request text parsed", "text", text)

		spoilersString := r.MultipartForm.Value["spoilers"]
		logger.Debug("request spoilers parsed", "spoilers", strings.Join(spoilersString, ","))

		files := r.MultipartForm.File["files"]

		if len(files) != len(spoilersString) {
			writeError(w, logger, http.StatusBadRequest, request.RequestID, "number of files and spoilers not equal", "number of files and spoilers not equal")
			return
		}

		spoilers := make([]bool, 0, len(spoilersString))
		for _, s := range spoilersString {
			spoilers = append(spoilers, trueFalseParse(s))
		}

		request.Text = text

		request.Channels = set.NewSet(r.MultipartForm.Value["channels"]...)
		logger.Debug("request channels parsed", "request_channels", strings.Join(r.MultipartForm.Value["channels"], ","))

		for i, f := range files {
			fileLogger := logger.With("file_name", f.Filename)

			fileLogger.Debug("processing file")

			fi, err := f.Open()
			if err != nil {
				writeError(w, fileLogger, http.StatusInternalServerError, request.RequestID, fmt.Sprintf("failed to open file from multipart: %s", f.Filename), "failed to open file from multipart")
				return
			}

			content, err := io.ReadAll(fi)
			fi.Close()
			if err != nil {
				writeError(w, fileLogger, http.StatusInternalServerError, request.RequestID, fmt.Sprintf("failed to read file from multipart: %s", f.Filename), "failed to read file from multipart")
				return
			}

			contentType := f.Header.Get("Content-Type")
			if contentType == "" {
				contentType = mimetype.Detect(content).String()
				fileLogger.Debug("content type empty, autodetection required")
			}

			request.Files = append(request.Files, &heroldo.File{Name: f.Filename, ContentType: contentType, Spoiler: spoilers[i], Content: content})
			fileLogger.Debug("file appended", "content_type", contentType, "spoilers", spoilers)
		}

		sentChannels, err := sender.Send(r.Context(), request)
		if err != nil {
			writeError(w, logger, http.StatusRequestTimeout, request.RequestID, "request cancelled", "request cancelled")
			return
		}

		logger.Info("request sent", "sent_channels", strings.Join(sentChannels, ","))

		writeSuccess(w, logger, http.StatusAccepted, request.RequestID, sentChannels, "request complete")
	})
}
