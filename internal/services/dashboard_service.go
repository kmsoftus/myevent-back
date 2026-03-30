package services

import (
	"context"
	"errors"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type DashboardService struct {
	events repositories.EventRepository
	guests repositories.GuestRepository
	gifts  repositories.GiftRepository
}

func NewDashboardService(events repositories.EventRepository, guests repositories.GuestRepository, gifts repositories.GiftRepository) *DashboardService {
	return &DashboardService{
		events: events,
		guests: guests,
		gifts:  gifts,
	}
}

func (s *DashboardService) GetByEvent(ctx context.Context, userID, eventID string) (*dto.DashboardResponse, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	guests, err := s.guests.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	response := &dto.DashboardResponse{
		GuestsTotal: len(guests),
	}

	for _, guest := range guests {
		switch guest.RSVPStatus {
		case "confirmed":
			response.GuestsConfirmed++
		case "declined":
			response.GuestsDeclined++
		default:
			response.GuestsPending++
		}

		if guest.CheckedInAt != nil {
			response.CheckedInTotal++
		}
	}

	gifts, err := s.gifts.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	response.GiftsTotal = len(gifts)
	for _, gift := range gifts {
		switch gift.Status {
		case "confirmed":
			response.GiftsConfirmed++
		case "pending_payment":
			response.GiftsPendingPayment++
		}
	}

	return response, nil
}

func (s *DashboardService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
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
