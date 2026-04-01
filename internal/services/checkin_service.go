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

type CheckInResult struct {
	Guest            *models.Guest
	AlreadyCheckedIn bool
	CompanionNames   []string
}

type CheckInService struct {
	events repositories.EventRepository
	guests repositories.GuestRepository
	rsvps  repositories.RSVPRepository
}

func NewCheckInService(events repositories.EventRepository, guests repositories.GuestRepository, rsvps repositories.RSVPRepository) *CheckInService {
	return &CheckInService{
		events: events,
		guests: guests,
		rsvps:  rsvps,
	}
}

func (s *CheckInService) CheckIn(ctx context.Context, userID, eventID string, input dto.CheckInRequest) (*CheckInResult, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	guest, err := s.resolveGuest(ctx, eventID, input)
	if err != nil {
		return nil, err
	}

	companionNames := s.fetchCompanionNames(ctx, guest.ID)

	if guest.CheckedInAt != nil {
		return &CheckInResult{Guest: guest, AlreadyCheckedIn: true, CompanionNames: companionNames}, nil
	}

	now := time.Now().UTC()
	guest.CheckedInAt = &now
	guest.UpdatedAt = now

	if err := s.guests.Update(ctx, guest); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: Identificadores do convidado ja estao em uso.", ErrConflict)
		}
		return nil, err
	}

	return &CheckInResult{Guest: guest, AlreadyCheckedIn: false, CompanionNames: companionNames}, nil
}

func (s *CheckInService) fetchCompanionNames(ctx context.Context, guestID string) []string {
	rsvp, err := s.rsvps.GetByGuestID(ctx, guestID)
	if err != nil {
		return []string{}
	}
	if rsvp.CompanionNames == nil {
		return []string{}
	}
	return rsvp.CompanionNames
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
		return nil, fmt.Errorf("%w: Informe qr_code_token ou guest_id, nao ambos.", ErrValidation)
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
