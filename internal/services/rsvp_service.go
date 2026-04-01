package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
	"myevent-back/internal/utils"
)

type RSVPDetails struct {
	RSVP  *models.RSVP
	Guest *models.Guest
}

type RSVPService struct {
	events repositories.EventRepository
	guests repositories.GuestRepository
	rsvps  repositories.RSVPRepository
}

func NewRSVPService(events repositories.EventRepository, guests repositories.GuestRepository, rsvps repositories.RSVPRepository) *RSVPService {
	return &RSVPService{
		events: events,
		guests: guests,
		rsvps:  rsvps,
	}
}

func (s *RSVPService) SubmitBySlug(ctx context.Context, slug string, input dto.CreateRSVPRequest) (*RSVPDetails, error) {
	event, err := s.events.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if event.Status != "published" {
		return nil, ErrForbidden
	}

	guestIdentifier := strings.TrimSpace(input.GuestIdentifier)
	if guestIdentifier == "" {
		return nil, fmt.Errorf("%w: Informe o identificador do convidado.", ErrValidation)
	}

	var guest *models.Guest

	if event.OpenRSVP {
		guest, err = s.findOrCreateOpenRSVPGuest(ctx, event.ID, guestIdentifier)
		if err != nil {
			return nil, err
		}
	} else {
		guest, err = s.guests.GetByInviteCode(ctx, guestIdentifier)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}

		if guest.EventID != event.ID {
			return nil, ErrNotFound
		}
	}

	status, companionsCount, err := validateRSVPInput(input.Status, input.CompanionsCount, guest.MaxCompanions)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rsvp := &models.RSVP{
		ID:              uuid.NewString(),
		EventID:         event.ID,
		GuestID:         guest.ID,
		Status:          status,
		CompanionsCount: companionsCount,
		Message:         strings.TrimSpace(input.Message),
		RespondedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.rsvps.Upsert(ctx, rsvp); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	guest.RSVPStatus = status
	guest.UpdatedAt = now
	if err := s.guests.Update(ctx, guest); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &RSVPDetails{
		RSVP:  rsvp,
		Guest: guest,
	}, nil
}

func (s *RSVPService) ListByEvent(ctx context.Context, userID, eventID string) ([]RSVPDetails, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	rsvps, err := s.rsvps.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	response := make([]RSVPDetails, 0, len(rsvps))
	for _, rsvp := range rsvps {
		guest, err := s.guests.GetByID(ctx, rsvp.GuestID)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				continue
			}
			return nil, err
		}

		response = append(response, RSVPDetails{
			RSVP:  rsvp,
			Guest: guest,
		})
	}

	return response, nil
}

func (s *RSVPService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
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

// findOrCreateOpenRSVPGuest looks up a guest by name for an open-RSVP event.
// If no guest with that exact name (case-insensitive) exists yet, one is created on the spot.
func (s *RSVPService) findOrCreateOpenRSVPGuest(ctx context.Context, eventID, name string) (*models.Guest, error) {
	guests, err := s.guests.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, g := range guests {
		if strings.ToLower(g.Name) == nameLower {
			return g, nil
		}
	}

	// Guest not found — create automatically
	now := time.Now().UTC()
	var created *models.Guest

	for attempt := 0; attempt < 5; attempt++ {
		guest := &models.Guest{
			ID:            uuid.NewString(),
			EventID:       eventID,
			Name:          name,
			InviteCode:    utils.RandomUpperString(8),
			QRCodeToken:   utils.RandomString(32),
			MaxCompanions: 10,
			RSVPStatus:    "pending",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := s.guests.Create(ctx, guest); err == nil {
			created = guest
			break
		} else if !errors.Is(err, repositories.ErrConflict) {
			return nil, err
		}
	}

	if created == nil {
		return nil, fmt.Errorf("%w: Nao foi possivel gerar identificadores para o convidado.", ErrConflict)
	}

	return created, nil
}

func validateRSVPInput(status string, companionsCount, maxCompanions int) (string, int, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "confirmed":
		if companionsCount < 0 {
			return "", 0, fmt.Errorf("%w: O numero de acompanhantes nao pode ser negativo.", ErrValidation)
		}
		if companionsCount > maxCompanions {
			return "", 0, fmt.Errorf("%w: O numero de acompanhantes ultrapassa o limite do convidado.", ErrValidation)
		}
		return status, companionsCount, nil
	case "declined":
		if companionsCount < 0 {
			return "", 0, fmt.Errorf("%w: O numero de acompanhantes nao pode ser negativo.", ErrValidation)
		}
		if companionsCount > 0 {
			return "", 0, fmt.Errorf("%w: Ao recusar o convite, o numero de acompanhantes deve ser 0.", ErrValidation)
		}
		return status, 0, nil
	default:
		return "", 0, fmt.Errorf("%w: Status de confirmacao invalido.", ErrValidation)
	}
}
