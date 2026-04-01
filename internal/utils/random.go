package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	alphaNumeric      = "abcdefghijklmnopqrstuvwxyz0123456789"
	alphaNumericUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	digits            = "0123456789"
)

func RandomString(length int) string {
	return randomFromCharset(length, alphaNumeric)
}

func RandomUpperString(length int) string {
	return randomFromCharset(length, alphaNumericUpper)
}

func RandomDigits(length int) string {
	return randomFromCharset(length, digits)
}

func NormalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
			lastDash = false
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}

func randomFromCharset(length int, charset string) string {
	if length <= 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(length)
	max := big.NewInt(int64(len(charset)))
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			builder.WriteByte(charset[i%len(charset)])
			continue
		}
		builder.WriteByte(charset[index.Int64()])
	}

	return builder.String()
}
