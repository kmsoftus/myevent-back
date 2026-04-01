package services

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxEventDescriptionLength         = 2000
	maxEventHostMessageLength         = 1000
	maxGiftDescriptionLength          = 2000
	maxGuestNotesLength               = 1000
	maxRSVPMessageLength              = 500
	maxGiftTransactionGuestContactLen = 255
	maxGiftTransactionMessageLength   = 500
)

func validateTextMaxLength(field, label, value string, max int, code string) error {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return nil
	}

	message := fmt.Sprintf("O campo %s deve ter no maximo %d caracteres.", label, max)
	return NewValidationError(
		message,
		code,
		FieldError{Field: field, Message: message},
	)
}
