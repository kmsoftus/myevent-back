package services

import (
	"context"
	"errors"
	"strings"

	"myevent-back/internal/repositories"
)

type AccountService struct {
	users   repositories.UserRepository
	events  repositories.EventRepository
	gifts   repositories.GiftRepository
	uploads *UploadService
}

func NewAccountService(
	users repositories.UserRepository,
	events repositories.EventRepository,
	gifts repositories.GiftRepository,
	uploads *UploadService,
) *AccountService {
	return &AccountService{
		users:   users,
		events:  events,
		gifts:   gifts,
		uploads: uploads,
	}
}

func (s *AccountService) Delete(ctx context.Context, userID, email string) error {
	userID = strings.TrimSpace(userID)
	email = normalizeEmail(email)
	if userID == "" {
		return NewUnauthorizedError(
			"Sessao invalida. Faca login novamente.",
			"auth_session_invalid",
		)
	}

	if email == "" {
		return NewValidationError(
			"Digite o e-mail da conta para confirmar a exclusao.",
			"auth_delete_email_required",
			FieldError{Field: "email", Message: "Digite o e-mail da conta para confirmar a exclusao."},
		)
	}
	if err := validateEmail(email); err != nil {
		return err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NewUnauthorizedError(
				"Sessao invalida. Faca login novamente.",
				"auth_session_invalid",
			)
		}
		return err
	}
	if normalizeEmail(user.Email) != email {
		return NewValidationError(
			"O e-mail digitado nao confere com a conta.",
			"auth_delete_email_mismatch",
			FieldError{Field: "email", Message: "O e-mail digitado nao confere com a conta."},
		)
	}

	managedKeys, err := s.collectManagedUploadKeys(ctx, userID)
	if err != nil {
		return err
	}

	for key := range managedKeys {
		if err := s.uploads.Delete(ctx, key); err != nil {
			return err
		}
	}

	if err := s.users.Delete(ctx, userID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NewUnauthorizedError(
				"Sessao invalida. Faca login novamente.",
				"auth_session_invalid",
			)
		}
		return err
	}

	return nil
}

func (s *AccountService) collectManagedUploadKeys(ctx context.Context, userID string) (map[string]struct{}, error) {
	keys := make(map[string]struct{})

	events, err := s.events.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		s.collectManagedKey(event.CoverImageURL, keys)

		gifts, err := s.gifts.ListByEventID(ctx, event.ID)
		if err != nil {
			return nil, err
		}

		for _, gift := range gifts {
			s.collectManagedKey(gift.ImageURL, keys)
		}
	}

	return keys, nil
}

func (s *AccountService) collectManagedKey(raw string, keys map[string]struct{}) {
	if s.uploads == nil {
		return
	}

	key, ok := s.uploads.ManagedKeyFromURL(raw)
	if !ok {
		return
	}

	keys[key] = struct{}{}
}
