package services

import (
	"fmt"
	"net/url"
	"strings"
)

func validateOptionalURL(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%w: O campo %s deve ser uma URL valida.", ErrValidation, field)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("%w: O campo %s deve usar http ou https.", ErrValidation, field)
	}
}
