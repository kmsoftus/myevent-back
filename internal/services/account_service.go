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

func (s *AccountService) Delete(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return NewUnauthorizedError(
			"Sessao invalida. Faca login novamente.",
			"auth_session_invalid",
		)
	}

	if _, err := s.users.GetByID(ctx, userID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NewUnauthorizedError(
				"Sessao invalida. Faca login novamente.",
				"auth_session_invalid",
			)
		}
		return err
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
