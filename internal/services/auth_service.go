package services

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/auth"
	"myevent-back/internal/mailer"
	"myevent-back/internal/models"
	"myevent-back/internal/notifier"
	"myevent-back/internal/repositories"
)

type AuthService struct {
	users               repositories.UserRepository
	passwordResetTokens repositories.PasswordResetTokenRepository
	jwt                 *auth.JWTManager
	passwordResetTTL    time.Duration
	passwordResetURL    string
	passwordResetSender mailer.PasswordResetSender
	registrationSender  notifier.RegistrationSender
}

func NewAuthService(
	users repositories.UserRepository,
	passwordResetTokens repositories.PasswordResetTokenRepository,
	jwt *auth.JWTManager,
	passwordResetTTL time.Duration,
	passwordResetURL string,
	passwordResetSender mailer.PasswordResetSender,
	registrationSender notifier.RegistrationSender,
) *AuthService {
	if passwordResetSender == nil {
		passwordResetSender = mailer.NoopSender{}
	}
	if registrationSender == nil {
		registrationSender = notifier.NoopRegistrationSender{}
	}

	return &AuthService{
		users:               users,
		passwordResetTokens: passwordResetTokens,
		jwt:                 jwt,
		passwordResetTTL:    passwordResetTTL,
		passwordResetURL:    passwordResetURL,
		passwordResetSender: passwordResetSender,
		registrationSender:  registrationSender,
	}
}

func (s *AuthService) Register(ctx context.Context, name, email, password string, attribution models.UserAttribution) (*models.User, string, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	password = strings.TrimSpace(password)
	attribution = normalizeUserAttribution(attribution)

	if name == "" {
		return nil, "", NewValidationError(
			"Informe seu nome.",
			"auth_name_required",
			FieldError{Field: "name", Message: "Informe seu nome."},
		)
	}
	if err := validateEmail(email); err != nil {
		return nil, "", err
	}
	if err := validatePassword(password); err != nil {
		return nil, "", err
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:           uuid.NewString(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Attribution:  attribution,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, "", NewConflictError(
				"Este e-mail ja esta cadastrado.",
				"auth_email_already_registered",
				FieldError{Field: "email", Message: "Este e-mail ja esta cadastrado."},
			)
		}
		return nil, "", err
	}

	token, err := s.jwt.GenerateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	if err := s.registrationSender.SendNewRegistration(ctx, notifier.NewRegistrationMessage{
		UserID:      user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Attribution: user.Attribution,
		CreatedAt:   user.CreatedAt,
	}); err != nil {
		log.Printf("telegram registration notification failed for user %s: %v", user.ID, err)
	}

	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	email = normalizeEmail(email)
	password = strings.TrimSpace(password)

	if err := validateEmail(email); err != nil {
		return nil, "", err
	}
	if password == "" {
		return nil, "", NewValidationError(
			"Informe sua senha.",
			"auth_password_required",
			FieldError{Field: "password", Message: "Informe sua senha."},
		)
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, "", NewInvalidCredentialsError(
				"E-mail ou senha invalidos.",
				"auth_invalid_credentials",
			)
		}
		return nil, "", err
	}

	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, "", NewInvalidCredentialsError(
			"E-mail ou senha invalidos.",
			"auth_invalid_credentials",
		)
	}

	token, err := s.jwt.GenerateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, NewUnauthorizedError(
			"Sessao invalida. Faca login novamente.",
			"auth_session_invalid",
		)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NewUnauthorizedError(
				"Sessao invalida. Faca login novamente.",
				"auth_session_invalid",
			)
		}
		return nil, err
	}

	return user, nil
}

func validateEmail(email string) error {
	if email == "" {
		return NewValidationError(
			"Informe seu e-mail.",
			"auth_email_required",
			FieldError{Field: "email", Message: "Informe seu e-mail."},
		)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return NewValidationError(
			"Informe um e-mail valido.",
			"auth_email_invalid",
			FieldError{Field: "email", Message: "Informe um e-mail valido."},
		)
	}
	return nil
}

func validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < 8 {
		return NewValidationError(
			"A senha deve ter pelo menos 8 caracteres.",
			"auth_password_too_short",
			FieldError{Field: "password", Message: "A senha deve ter pelo menos 8 caracteres."},
		)
	}

	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeUserAttribution(attribution models.UserAttribution) models.UserAttribution {
	return models.UserAttribution{
		UTMSource:   strings.TrimSpace(attribution.UTMSource),
		UTMMedium:   strings.TrimSpace(attribution.UTMMedium),
		UTMCampaign: strings.TrimSpace(attribution.UTMCampaign),
		UTMTerm:     strings.TrimSpace(attribution.UTMTerm),
		UTMContent:  strings.TrimSpace(attribution.UTMContent),
	}
}
