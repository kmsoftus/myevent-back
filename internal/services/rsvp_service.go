package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	events                       repositories.EventRepository
	guests                       repositories.GuestRepository
	rsvps                        repositories.RSVPRepository
	openRSVPGuests               atomicOpenRSVPGuestRepository
	openRSVPDefaultMaxCompanions int
	organizerNotifications       *OrganizerNotificationService
}

func NewRSVPService(
	events repositories.EventRepository,
	guests repositories.GuestRepository,
	rsvps repositories.RSVPRepository,
	openRSVPDefaultMaxCompanions int,
	organizerNotifications ...*OrganizerNotificationService,
) *RSVPService {
	service := &RSVPService{
		events:                       events,
		guests:                       guests,
		rsvps:                        rsvps,
		openRSVPDefaultMaxCompanions: max(0, openRSVPDefaultMaxCompanions),
	}
	if len(organizerNotifications) > 0 {
		service.organizerNotifications = organizerNotifications[0]
	}

	if openRSVPGuests, ok := guests.(atomicOpenRSVPGuestRepository); ok {
		service.openRSVPGuests = openRSVPGuests
	}

	return service
}

// GuestCandidate is a lightweight guest representation returned by the search endpoint.
type GuestCandidate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortCode string `json:"short_code"`
}

// SearchGuestsBySlug returns guests whose names contain the query string (case-insensitive).
// Used by the frontend to let the guest confirm their identity before submitting RSVP.
func (s *RSVPService) SearchGuestsBySlug(ctx context.Context, slug, query string) ([]GuestCandidate, error) {
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

	query = strings.TrimSpace(query)
	if query == "" {
		return []GuestCandidate{}, nil
	}

	guests, err := s.guests.SearchByName(ctx, event.ID, query, 10)
	if err != nil {
		return nil, err
	}

	candidates := make([]GuestCandidate, 0, len(guests))
	for _, g := range guests {
		candidates = append(candidates, GuestCandidate{
			ID:        g.ID,
			Name:      g.Name,
			ShortCode: g.ShortCode,
		})
	}
	return candidates, nil
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
		// Accept short code (digits only, 6-7 chars) or legacy invite code
		isShortCode := isDigitsOnly(guestIdentifier) && len(guestIdentifier) >= 6 && len(guestIdentifier) <= 7
		if isShortCode {
			guest, err = s.guests.GetByShortCode(ctx, event.ID, guestIdentifier)
		} else {
			guest, err = s.guests.GetByInviteCode(ctx, guestIdentifier)
		}
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

	status, companionsCount, companionNames, err := validateRSVPInput(input.Status, input.CompanionsCount, input.CompanionNames, guest.MaxCompanions)
	if err != nil {
		return nil, err
	}
	if err := validateTextMaxLength("message", "mensagem", input.Message, maxRSVPMessageLength, "rsvp_message_too_long"); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rsvp := &models.RSVP{
		ID:              uuid.NewString(),
		EventID:         event.ID,
		GuestID:         guest.ID,
		Status:          status,
		CompanionsCount: companionsCount,
		CompanionNames:  companionNames,
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

	if s.organizerNotifications != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), asyncNotificationTimeout)
			defer cancel()

			if err := s.organizerNotifications.NotifyNewRSVP(notifyCtx, event, guest, rsvp); err != nil {
				log.Printf(
					"organizer RSVP push notification failed for event %s guest %s: %v",
					event.ID,
					guest.ID,
					err,
				)
			}
		}()
	}

	return &RSVPDetails{
		RSVP:  rsvp,
		Guest: guest,
	}, nil
}

func (s *RSVPService) ListByEvent(ctx context.Context, userID, eventID string, page, pageSize int) (*PagedResult[RSVPDetails], error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	pagination := normalizePagination(page, pageSize)

	total, err := s.rsvps.CountByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	rsvps, err := s.rsvps.ListByEventIDPaged(ctx, eventID, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, err
	}

	// Batch-load guests in a single query instead of one query per RSVP.
	guestIDs := make([]string, len(rsvps))
	for i, rsvp := range rsvps {
		guestIDs[i] = rsvp.GuestID
	}
	guestSlice, err := s.guests.GetByIDs(ctx, guestIDs)
	if err != nil {
		return nil, err
	}
	guestByID := make(map[string]*models.Guest, len(guestSlice))
	for _, g := range guestSlice {
		guestByID[g.ID] = g
	}

	response := make([]RSVPDetails, 0, len(rsvps))
	for _, rsvp := range rsvps {
		guest, ok := guestByID[rsvp.GuestID]
		if !ok {
			continue
		}
		response = append(response, RSVPDetails{
			RSVP:  rsvp,
			Guest: guest,
		})
	}

	return &PagedResult[RSVPDetails]{
		Items:      response,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages(total, pagination.PageSize),
	}, nil
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
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: Informe o nome do convidado.", ErrValidation)
	}

	if s.openRSVPGuests != nil {
		now := time.Now().UTC()
		for attempt := 0; attempt < 5; attempt++ {
			guest := &models.Guest{
				ID:            uuid.NewString(),
				EventID:       eventID,
				Name:          name,
				InviteCode:    utils.RandomUpperString(8),
				ShortCode:     utils.RandomDigits(6),
				QRCodeToken:   utils.RandomString(32),
				MaxCompanions: s.openRSVPDefaultMaxCompanions,
				RSVPStatus:    "pending",
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			created, err := s.openRSVPGuests.FindOrCreateOpenRSVPGuest(ctx, guest)
			if err == nil {
				return created, nil
			}
			if !errors.Is(err, repositories.ErrConflict) {
				return nil, err
			}
		}

		return nil, fmt.Errorf("%w: Nao foi possivel gerar identificadores para o convidado.", ErrConflict)
	}

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
			ShortCode:     utils.RandomDigits(6),
			QRCodeToken:   utils.RandomString(32),
			MaxCompanions: s.openRSVPDefaultMaxCompanions,
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

// RSVPLookupResult holds the data returned by a public code lookup.
type RSVPLookupResult struct {
	GuestName      string `json:"guest_name"`
	GuestShortCode string `json:"guest_short_code"`
	QRCodeToken    string `json:"qr_code_token"`
	RSVPStatus     string `json:"rsvp_status"`
}

// LookupBySlug retrieves the current RSVP status for a guest identified by short code.
// It is used by the public RSVP page to detect already-confirmed guests and skip the form.
func (s *RSVPService) LookupBySlug(ctx context.Context, slug, code string) (*RSVPLookupResult, error) {
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

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("%w: Informe o código do convite.", ErrValidation)
	}

	var guest *models.Guest
	if isDigitsOnly(code) && len(code) >= 6 && len(code) <= 7 {
		guest, err = s.guests.GetByShortCode(ctx, event.ID, code)
	} else {
		guest, err = s.guests.GetByInviteCode(ctx, code)
	}
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if guest.EventID != event.ID {
		return nil, ErrNotFound
	}

	return &RSVPLookupResult{
		GuestName:      guest.Name,
		GuestShortCode: guest.ShortCode,
		QRCodeToken:    guest.QRCodeToken,
		RSVPStatus:     guest.RSVPStatus,
	}, nil
}

func isDigitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateRSVPInput(status string, companionsCount int, companionNames []string, maxCompanions int) (string, int, []string, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "confirmed":
		if companionsCount < 0 {
			return "", 0, nil, fmt.Errorf("%w: O numero de acompanhantes nao pode ser negativo.", ErrValidation)
		}
		if companionsCount > maxCompanions {
			return "", 0, nil, fmt.Errorf("%w: O numero de acompanhantes ultrapassa o limite do convidado.", ErrValidation)
		}
		// Normalize companion names: trim, drop empties, cap at companionsCount
		names := make([]string, 0, companionsCount)
		for _, n := range companionNames {
			n = strings.TrimSpace(n)
			if n != "" {
				names = append(names, n)
			}
		}
		if len(names) > companionsCount {
			names = names[:companionsCount]
		}
		return status, companionsCount, names, nil
	case "declined":
		if companionsCount > 0 {
			return "", 0, nil, fmt.Errorf("%w: Ao recusar o convite, o numero de acompanhantes deve ser 0.", ErrValidation)
		}
		return status, 0, []string{}, nil
	default:
		return "", 0, nil, fmt.Errorf("%w: Status de confirmacao invalido.", ErrValidation)
	}
}
