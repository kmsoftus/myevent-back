package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strings"

	"myevent-back/internal/services"
)

type errorResponse struct {
	Message string                `json:"message"`
	Error   string                `json:"error"`
	Code    string                `json:"code,omitempty"`
	Details []services.FieldError `json:"details,omitempty"`
}

func DecodeJSON(w nethttp.ResponseWriter, r *nethttp.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		WriteErrorResponse(w, nethttp.StatusBadRequest, "Corpo da requisicao invalido.", "invalid_json", nil)
		return false
	}

	if decoder.More() {
		WriteErrorResponse(w, nethttp.StatusBadRequest, "Corpo da requisicao invalido.", "invalid_json", nil)
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
	WriteErrorResponse(w, status, message, "", nil)
}

func WriteErrorResponse(w nethttp.ResponseWriter, status int, message, code string, details []services.FieldError) {
	WriteJSON(w, status, errorResponse{
		Message: message,
		Error:   message,
		Code:    code,
		Details: details,
	})
}

func MapError(w nethttp.ResponseWriter, err error) {
	var appErr *services.AppError

	switch {
	case err == nil:
		return
	case errors.As(err, &appErr):
		WriteErrorResponse(w, statusFromError(err), appErr.Message, appErr.Code, appErr.Details)
	case errors.Is(err, services.ErrValidation):
		WriteErrorResponse(w, nethttp.StatusBadRequest, extractServiceErrorMessage(err), "validation_error", nil)
	case errors.Is(err, services.ErrUnauthorized), errors.Is(err, services.ErrInvalidCredentials):
		WriteErrorResponse(w, nethttp.StatusUnauthorized, extractServiceErrorMessage(err), "unauthorized", nil)
	case errors.Is(err, services.ErrForbidden):
		WriteErrorResponse(w, nethttp.StatusForbidden, extractServiceErrorMessage(err), "forbidden", nil)
	case errors.Is(err, services.ErrNotFound):
		WriteErrorResponse(w, nethttp.StatusNotFound, extractServiceErrorMessage(err), "not_found", nil)
	case errors.Is(err, services.ErrConflict):
		WriteErrorResponse(w, nethttp.StatusConflict, extractServiceErrorMessage(err), "conflict", nil)
	default:
		WriteErrorResponse(w, nethttp.StatusInternalServerError, "Erro interno do servidor.", "internal_server_error", nil)
	}
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, services.ErrValidation):
		return nethttp.StatusBadRequest
	case errors.Is(err, services.ErrUnauthorized), errors.Is(err, services.ErrInvalidCredentials):
		return nethttp.StatusUnauthorized
	case errors.Is(err, services.ErrForbidden):
		return nethttp.StatusForbidden
	case errors.Is(err, services.ErrNotFound):
		return nethttp.StatusNotFound
	case errors.Is(err, services.ErrConflict):
		return nethttp.StatusConflict
	default:
		return nethttp.StatusInternalServerError
	}
}

func extractServiceErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	prefixes := []string{
		services.ErrValidation.Error() + ": ",
		services.ErrUnauthorized.Error() + ": ",
		services.ErrForbidden.Error() + ": ",
		services.ErrNotFound.Error() + ": ",
		services.ErrConflict.Error() + ": ",
		services.ErrInvalidCredentials.Error() + ": ",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(message, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(message, prefix))
		}
	}

	return message
}
