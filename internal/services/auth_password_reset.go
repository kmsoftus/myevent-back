package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/auth"
	"myevent-back/internal/mailer"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
	"myevent-back/internal/utils"
)

const forgotPasswordSuccessMessage = "Se o e-mail existir, enviaremos as instrucoes para recuperar sua senha."

func (s *AuthService) ForgotPassword(ctx context.Context, email string) (string, error) {
	email = normalizeEmail(email)

	if err := validateEmail(email); err != nil {
		return "", err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == repositories.ErrNotFound {
			return forgotPasswordSuccessMessage, nil
		}
		return "", err
	}

	now := time.Now().UTC()
	if err := s.passwordResetTokens.DeleteActiveByUserID(ctx, user.ID, now); err != nil {
		return "", err
	}

	rawToken, tokenHash := newPasswordResetToken()
	token := &models.PasswordResetToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(s.passwordResetTTL),
		CreatedAt: now,
	}

	if err := s.passwordResetTokens.Create(ctx, token); err != nil {
		return "", err
	}

	resetURL, err := buildPasswordResetURL(s.passwordResetURL, rawToken)
	if err != nil {
		return "", err
	}

	if err := s.passwordResetSender.SendPasswordReset(ctx, mailer.PasswordResetMessage{
		ExpiresIn: s.passwordResetTTL,
		ResetURL:  resetURL,
		ToEmail:   user.Email,
		ToName:    user.Name,
	}); err != nil {
		return "", err
	}

	return forgotPasswordSuccessMessage, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, password string) (string, error) {
	token = strings.TrimSpace(token)
	password = strings.TrimSpace(password)

	if token == "" {
		return "", NewValidationError(
			"Informe o token de recuperacao.",
			"auth_reset_token_required",
			FieldError{Field: "token", Message: "Informe o token de recuperacao."},
		)
	}
	if err := validatePassword(password); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	resetToken, err := s.passwordResetTokens.Consume(ctx, hashPasswordResetToken(token), now)
	if err != nil {
		if err == repositories.ErrNotFound {
			return "", NewValidationError(
				"Este link de recuperacao e invalido ou expirou.",
				"auth_reset_token_invalid",
				FieldError{Field: "token", Message: "Este link de recuperacao e invalido ou expirou."},
			)
		}
		return "", err
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}

	if err := s.users.UpdatePassword(ctx, resetToken.UserID, passwordHash, now); err != nil {
		if err == repositories.ErrNotFound {
			return "", NewValidationError(
				"Este link de recuperacao e invalido ou expirou.",
				"auth_reset_token_invalid",
				FieldError{Field: "token", Message: "Este link de recuperacao e invalido ou expirou."},
			)
		}
		return "", err
	}

	return "Senha redefinida com sucesso.", nil
}

func newPasswordResetToken() (rawToken, tokenHash string) {
	rawToken = utils.RandomString(48)
	return rawToken, hashPasswordResetToken(rawToken)
}

func hashPasswordResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func buildPasswordResetURL(baseURL, rawToken string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse password reset url: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid password reset url")
	}

	query := parsedURL.Query()
	query.Set("token", rawToken)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}
