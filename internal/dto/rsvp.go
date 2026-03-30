package dto

import (
	"time"

	"myevent-back/internal/models"
)

type CreateRSVPRequest struct {
	GuestIdentifier string `json:"guest_identifier"`
	Status          string `json:"status"`
	CompanionsCount int    `json:"companions_count"`
	Message         string `json:"message"`
}

type RSVPResponse struct {
	ID              string    `json:"id"`
	EventID         string    `json:"event_id"`
	GuestID         string    `json:"guest_id"`
	GuestName       string    `json:"guest_name"`
	Status          string    `json:"status"`
	CompanionsCount int       `json:"companions_count"`
	Message         string    `json:"message,omitempty"`
	RespondedAt     time.Time `json:"responded_at"`
}

func NewRSVPResponse(rsvp *models.RSVP, guest *models.Guest) RSVPResponse {
	return RSVPResponse{
		ID:              rsvp.ID,
		EventID:         rsvp.EventID,
		GuestID:         rsvp.GuestID,
		GuestName:       guest.Name,
		Status:          rsvp.Status,
		CompanionsCount: rsvp.CompanionsCount,
		Message:         rsvp.Message,
		RespondedAt:     rsvp.RespondedAt,
	}
}
