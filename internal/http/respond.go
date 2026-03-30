package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"

	"myevent-back/internal/services"
)

type errorResponse struct {
	Error string `json:"error"`
}

func DecodeJSON(w nethttp.ResponseWriter, r *nethttp.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		WriteError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return false
	}

	if decoder.More() {
		WriteError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return false
	}

	return true
}

func WriteJSON(w nethttp.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w nethttp.ResponseWriter, status int, message string) {
	WriteJSON(w, status, errorResponse{Error: message})
}

func MapError(w nethttp.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, services.ErrValidation):
		WriteError(w, nethttp.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrUnauthorized), errors.Is(err, services.ErrInvalidCredentials):
		WriteError(w, nethttp.StatusUnauthorized, err.Error())
	case errors.Is(err, services.ErrForbidden):
		WriteError(w, nethttp.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrNotFound):
		WriteError(w, nethttp.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrConflict):
		WriteError(w, nethttp.StatusConflict, err.Error())
	default:
		WriteError(w, nethttp.StatusInternalServerError, "internal server error")
	}
}
