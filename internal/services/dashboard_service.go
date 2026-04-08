package services

import (
	"context"
	"errors"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

// guestStatsAggregator is an optional interface that postgres repos implement
// to compute dashboard counters in a single SQL aggregate query.
type guestStatsAggregator interface {
	GuestStatsByEventID(ctx context.Context, eventID string) (repositories.GuestDashboardStats, error)
}

type giftStatsAggregator interface {
	GiftStatsByEventID(ctx context.Context, eventID string) (repositories.GiftDashboardStats, error)
}

type DashboardService struct {
	users  repositories.UserRepository
	events repositories.EventRepository
	guests repositories.GuestRepository
	gifts  repositories.GiftRepository
}

func NewDashboardService(users repositories.UserRepository, events repositories.EventRepository, guests repositories.GuestRepository, gifts repositories.GiftRepository) *DashboardService {
	return &DashboardService{
		users:  users,
		events: events,
		guests: guests,
		gifts:  gifts,
	}
}

func (s *DashboardService) GetByEvent(ctx context.Context, userID, eventID string) (*dto.DashboardResponse, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	response := &dto.DashboardResponse{
		User: dto.DashboardUserResponse{
			ID:              user.ID,
			Name:            user.Name,
			ProfilePhotoURL: user.ProfilePhotoURL,
		},
	}

	// Use SQL aggregation if available (postgres), fall back to in-memory loop otherwise (tests).
	if agg, ok := s.guests.(guestStatsAggregator); ok {
		stats, err := agg.GuestStatsByEventID(ctx, eventID)
		if err != nil {
			return nil, err
		}
		response.GuestsTotal = stats.Total
		response.GuestsConfirmed = stats.Confirmed
		response.GuestsDeclined = stats.Declined
		response.GuestsPending = stats.Pending
		response.CheckedInTotal = stats.CheckedIn
	} else {		guests, err := s.guests.ListByEventID(ctx, eventID)
		if err != nil {
			return nil, err
		}
		response.GuestsTotal = len(guests)
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
	}

	if agg, ok := s.gifts.(giftStatsAggregator); ok {
		stats, err := agg.GiftStatsByEventID(ctx, eventID)
		if err != nil {
			return nil, err
		}
		response.GiftsTotal = stats.Total
		response.GiftsConfirmed = stats.Confirmed
		response.GiftsPendingPayment = stats.PendingPayment
	} else {
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
