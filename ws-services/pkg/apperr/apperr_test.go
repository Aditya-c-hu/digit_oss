package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestResolveBadRequest(t *testing.T) {
	status, code, msg := Resolve(BadRequest("EG_WS_X", "bad"))
	if status != http.StatusBadRequest || code != "EG_WS_X" || msg != "bad" {
		t.Errorf("got %d/%s/%s", status, code, msg)
	}
}

func TestResolveInternal(t *testing.T) {
	status, code, _ := Resolve(Internal("EG_WS_Y", "boom"))
	if status != http.StatusInternalServerError || code != "EG_WS_Y" {
		t.Errorf("got %d/%s", status, code)
	}
}

func TestResolvePlainError(t *testing.T) {
	status, code, _ := Resolve(errors.New("raw"))
	if status != http.StatusInternalServerError || code != "INTERNAL_SERVER_ERROR" {
		t.Errorf("got %d/%s", status, code)
	}
}

func TestResolveWrapped(t *testing.T) {
	wrapped := fmt.Errorf("ctx: %w", BadRequest("EG_WS_Z", "nested"))
	status, code, _ := Resolve(wrapped)
	if status != http.StatusBadRequest || code != "EG_WS_Z" {
		t.Errorf("wrapped apperr not resolved: %d/%s", status, code)
	}
}
