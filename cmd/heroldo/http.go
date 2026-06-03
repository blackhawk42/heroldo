package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/r/blackhawk42/heroldo/pkg/heroldo"
)

type SuccessResponse struct {
	ResponseCode int      `json:"response_code"`
	RequestID    string   `json:"request_id"`
	Channels     []string `json:"channels"`
}

type ErrorResponse struct {
	ResponseCode int    `json:"response_code"`
	RequestID    string `json:"request_id"`
	Error        string `json:"error"`
}

func trueFalseParse(s string) bool {
	if s == "true" {
		return true
	} else {
		return false
	}
}

func joinTexts(txts []string) string {
	return strings.Join(txts, "\n")
}

func RequestHandler(maxBodySize int64, sender *DiscordSender) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		request := heroldo.Request{}

		var err error
		request.RequestID, err = gonanoid.New()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusInternalServerError, Error: "failed to generate ID for request"})
			w.Write(response)

			slog.Error("failed to generate ID for request")
			return
		}

		logger := slog.With("request_id", request.RequestID)

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		err = r.ParseMultipartForm(maxBodySize)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusRequestEntityTooLarge, Error: "request body too large"})
				w.Write(response)

				logger.Error("request body too large", "error", err)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusBadRequest, Error: fmt.Sprintf("request failed to be parsed as a multiform request: %s", err)})
			w.Write(response)

			logger.Error("request failed to be parsed as a multiform request", "error", err)
			return
		}
		logger.Debug("multiform parsed")
		defer r.MultipartForm.RemoveAll()

		// Empty text and spolers are acceptable, so no ok checking.
		text := joinTexts(r.MultipartForm.Value["text"])
		spoilersString := r.MultipartForm.Value["spoilers"]
		files := r.MultipartForm.File["files"]

		if len(files) != len(spoilersString) {
			w.WriteHeader(http.StatusBadRequest)
			response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusBadRequest, Error: "number of files and spoilers not equal"})
			w.Write(response)

			logger.Error("number of files and spoilers not equal")
			return
		}

		contentTypes := r.MultipartForm.Value["content_types"]
		if len(files) != len(contentTypes) {
			w.WriteHeader(http.StatusBadRequest)
			response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusBadRequest, Error: "number of files and content_types not equal"})
			w.Write(response)

			logger.Error("number of files and content_types not equal")
			return
		}

		spoilers := make([]bool, 0, len(spoilersString))
		for _, s := range spoilersString {
			spoilers = append(spoilers, trueFalseParse(s))
		}

		request.Text = text

		for i, f := range files {
			fi, err := f.Open()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusInternalServerError, Error: fmt.Sprintf("failed to open file from multipart: %s", f.Filename)})
				w.Write(response)
				return
			}

			content, err := io.ReadAll(fi)
			fi.Close()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				response, _ := json.Marshal(ErrorResponse{ResponseCode: http.StatusInternalServerError, Error: fmt.Sprintf("failed to read file from multipart: %s", f.Filename)})
				w.Write(response)
				return
			}

			request.Files = append(request.Files, &heroldo.File{Name: f.Filename, ContentType: contentTypes[i], Spoiler: spoilers[i], Content: content})
		}

		logger.Info("sending request")

		if err := sender.SendContext(r.Context(), request); err != nil {
			w.WriteHeader(http.StatusRequestTimeout)
			response, _ := json.Marshal(ErrorResponse{
				ResponseCode: http.StatusRequestTimeout,
				RequestID:    request.RequestID,
				Error:        "request cancelled",
			})
			w.Write(response)

			logger.Info("request cancelled")
			return
		}

		w.WriteHeader(http.StatusOK)
		response, _ := json.Marshal(SuccessResponse{ResponseCode: http.StatusOK, RequestID: request.RequestID, Channels: sender.Channels()})
		w.Write(response)
	})
}
