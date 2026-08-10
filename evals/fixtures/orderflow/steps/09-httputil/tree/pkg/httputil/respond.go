package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"orderflow/pkg/xerrors"
)

// maxRequestBody is generous for our payloads and small enough to keep a bad
// client from filling memory.
const maxRequestBody = 1 << 20

// ErrEmptyBody is returned by Decode when the request carried no payload.
var ErrEmptyBody = errors.New("empty request body")

// ErrorBody is the envelope every endpoint uses for failures.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the machine readable part of a failure.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON serialises v before touching the response, so an encoding failure
// cannot leave a half written body behind.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// WriteError answers with the error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	body, err := json.Marshal(ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
	if err != nil {
		http.Error(w, message, status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// Decode reads a JSON request body into dst. Unknown fields are rejected: a
// client that misspells a field should hear about it instead of silently
// sending nothing.
func Decode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxRequestBody))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)
	switch {
	case errors.Is(err, io.EOF):
		return ErrEmptyBody
	case err != nil:
		return xerrors.Wrap(err, "decode request body")
	}
	if decoder.More() {
		return errors.New("request body holds more than one JSON value")
	}
	return nil
}
