package services

import "errors"

var (
	ErrValidation         = errors.New("validation error")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type AppError struct {
	Kind    error
	Message string
	Code    string
	Details []FieldError
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "erro desconhecido"
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Kind
}

func NewAppError(kind error, message, code string, details ...FieldError) *AppError {
	return &AppError{
		Kind:    kind,
		Message: message,
		Code:    code,
		Details: details,
	}
}

func NewValidationError(message, code string, details ...FieldError) error {
	return NewAppError(ErrValidation, message, code, details...)
}

func NewUnauthorizedError(message, code string, details ...FieldError) error {
	return NewAppError(ErrUnauthorized, message, code, details...)
}

func NewConflictError(message, code string, details ...FieldError) error {
	return NewAppError(ErrConflict, message, code, details...)
}

func NewInvalidCredentialsError(message, code string, details ...FieldError) error {
	return NewAppError(ErrInvalidCredentials, message, code, details...)
}
