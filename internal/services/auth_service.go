package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/auth"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type AuthService struct {
	users repositories.UserRepository
	jwt   *auth.JWTManager
}

func NewAuthService(users repositories.UserRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{
		users: users,
		jwt:   jwt,
	}
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*models.User, string, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	password = strings.TrimSpace(password)

	if name == "" {
		return nil, "", fmt.Errorf("%w: name is required", ErrValidation)
	}
	if err := validateEmail(email); err != nil {
		return nil, "", err
	}
	if len(password) < 8 {
		return nil, "", fmt.Errorf("%w: password must have at least 8 characters", ErrValidation)
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
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, "", fmt.Errorf("%w: email already registered", ErrConflict)
		}
		return nil, "", err
	}

	token, err := s.jwt.GenerateToken(user.ID)
	if err != nil {
		return nil, "", err
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
		return nil, "", fmt.Errorf("%w: password is required", ErrValidation)
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, "", ErrInvalidCredentials
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
		return nil, ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	return user, nil
}

func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("%w: email is required", ErrValidation)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email", ErrValidation)
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
