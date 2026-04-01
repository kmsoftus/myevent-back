package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
	"myevent-back/internal/utils"
)

type GuestService struct {
	events repositories.EventRepository
	guests repositories.GuestRepository
}

func NewGuestService(events repositories.EventRepository, guests repositories.GuestRepository) *GuestService {
	return &GuestService{
		events: events,
		guests: guests,
	}
}

func (s *GuestService) Create(ctx context.Context, userID, eventID string, input dto.CreateGuestRequest) (*models.Guest, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}
	if err := validateGuestPayload(input.Name, input.Email, input.MaxCompanions); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var created *models.Guest

	for attempt := 0; attempt < 5; attempt++ {
		guest := &models.Guest{
			ID:            uuid.NewString(),
			EventID:       eventID,
			Name:          strings.TrimSpace(input.Name),
			Email:         normalizeOptionalEmail(input.Email),
			Phone:         strings.TrimSpace(input.Phone),
			InviteCode:    utils.RandomUpperString(8),
			ShortCode:     utils.RandomDigits(6),
			QRCodeToken:   utils.RandomString(32),
			MaxCompanions: input.MaxCompanions,
			RSVPStatus:    "pending",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		err := s.guests.Create(ctx, guest)
		if err == nil {
			created = guest
			break
		}
		if !errors.Is(err, repositories.ErrConflict) {
			return nil, err
		}
	}

	if created == nil {
		return nil, fmt.Errorf("%w: Nao foi possivel gerar identificadores para o convidado.", ErrConflict)
	}

	return created, nil
}

func (s *GuestService) ListByEvent(ctx context.Context, userID, eventID string) ([]*models.Guest, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	return s.guests.ListByEventID(ctx, eventID)
}

func (s *GuestService) GetByID(ctx context.Context, userID, eventID, guestID string) (*models.Guest, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	guest, err := s.guests.GetByID(ctx, guestID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if guest.EventID != eventID {
		return nil, ErrNotFound
	}

	return guest, nil
}

func (s *GuestService) Update(ctx context.Context, userID, eventID, guestID string, input dto.UpdateGuestRequest) (*models.Guest, error) {
	guest, err := s.GetByID(ctx, userID, eventID, guestID)
	if err != nil {
		return nil, err
	}

	nextName := coalesceString(input.Name, guest.Name)
	nextEmail := coalesceString(input.Email, guest.Email)
	nextPhone := coalesceString(input.Phone, guest.Phone)
	nextMaxCompanions := coalesceInt(input.MaxCompanions, guest.MaxCompanions)

	if err := validateGuestPayload(nextName, nextEmail, nextMaxCompanions); err != nil {
		return nil, err
	}

	guest.Name = strings.TrimSpace(nextName)
	guest.Email = normalizeOptionalEmail(nextEmail)
	guest.Phone = strings.TrimSpace(nextPhone)
	guest.MaxCompanions = nextMaxCompanions
	guest.UpdatedAt = time.Now().UTC()

	if err := s.guests.Update(ctx, guest); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: Identificadores do convidado ja estao em uso.", ErrConflict)
		}
		return nil, err
	}

	return guest, nil
}

func (s *GuestService) Delete(ctx context.Context, userID, eventID, guestID string) error {
	guest, err := s.GetByID(ctx, userID, eventID, guestID)
	if err != nil {
		return err
	}

	if err := s.guests.Delete(ctx, guest.ID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (s *GuestService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if event.UserID != userID {
		return nil, ErrForbidden
	}

	return event, nil
}

func validateGuestPayload(name, email string, maxCompanions int) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: Informe o nome do convidado.", ErrValidation)
	}
	if maxCompanions < 0 {
		return fmt.Errorf("%w: O numero de acompanhantes nao pode ser negativo.", ErrValidation)
	}
	email = normalizeOptionalEmail(email)
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("%w: E-mail invalido.", ErrValidation)
		}
	}
	return nil
}

func normalizeOptionalEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func coalesceInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
