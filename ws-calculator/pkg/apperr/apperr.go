// Package apperr carries an eGov-style error code + HTTP status alongside a Go
// error, so transport handlers can emit the platform ResponseInfo/Errors
// envelope with the right status (400 for client/business, 500 for infra).
package apperr

import (
	"errors"
	"net/http"
)

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// BadRequest is a 400 — validation / business-rule failure.
func BadRequest(code, message string) *Error {
	return &Error{Code: code, Message: message, Status: http.StatusBadRequest}
}

// Internal is a 500 — unexpected server/infra failure.
func Internal(code, message string) *Error {
	return &Error{Code: code, Message: message, Status: http.StatusInternalServerError}
}

// Resolve extracts code + status from err, defaulting to 500/INTERNAL_SERVER_ERROR.
func Resolve(err error) (status int, code, message string) {
	var ae *Error
	if errors.As(err, &ae) {
		return ae.Status, ae.Code, ae.Message
	}
	return http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error()
}
