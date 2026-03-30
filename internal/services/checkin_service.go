package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type CheckInService struct {
	events repositories.EventRepository
	guests repositories.GuestRepository
}

func NewCheckInService(events repositories.EventRepository, guests repositories.GuestRepository) *CheckInService {
	return &CheckInService{
		events: events,
		guests: guests,
	}
}

func (s *CheckInService) CheckIn(ctx context.Context, userID, eventID string, input dto.CheckInRequest) (*models.Guest, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	guest, err := s.resolveGuest(ctx, eventID, input)
	if err != nil {
		return nil, err
	}

	if guest.CheckedInAt != nil {
		return nil, fmt.Errorf("%w: guest already checked in", ErrConflict)
	}

	now := time.Now().UTC()
	guest.CheckedInAt = &now
	guest.UpdatedAt = now

	if err := s.guests.Update(ctx, guest); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: guest identifiers already in use", ErrConflict)
		}
		return nil, err
	}

	return guest, nil
}

func (s *CheckInService) ListGuests(ctx context.Context, userID, eventID string) ([]*models.Guest, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	return s.guests.ListByEventID(ctx, eventID)
}

func (s *CheckInService) resolveGuest(ctx context.Context, eventID string, input dto.CheckInRequest) (*models.Guest, error) {
	qrCodeToken := strings.TrimSpace(input.QRCodeToken)
	guestID := strings.TrimSpace(input.GuestID)

	if (qrCodeToken == "" && guestID == "") || (qrCodeToken != "" && guestID != "") {
		return nil, fmt.Errorf("%w: provide either qr_code_token or guest_id", ErrValidation)
	}

	var (
		guest *models.Guest
		err   error
	)
	if qrCodeToken != "" {
		guest, err = s.guests.GetByQRCodeToken(ctx, qrCodeToken)
	} else {
		guest, err = s.guests.GetByID(ctx, guestID)
	}
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

func (s *CheckInService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
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
