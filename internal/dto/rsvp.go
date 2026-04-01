package dto

import (
	"time"

	"myevent-back/internal/models"
)

type CreateRSVPRequest struct {
	GuestIdentifier string   `json:"guest_identifier"`
	Status          string   `json:"status"`
	CompanionsCount int      `json:"companions_count"`
	CompanionNames  []string `json:"companion_names"`
	Message         string   `json:"message"`
}

type RSVPResponse struct {
	ID              string    `json:"id"`
	EventID         string    `json:"event_id"`
	GuestID         string    `json:"guest_id"`
	GuestName       string    `json:"guest_name"`
	GuestShortCode  string    `json:"guest_short_code"`
	QRCodeToken     string    `json:"qr_code_token"`
	Status          string    `json:"status"`
	CompanionsCount int       `json:"companions_count"`
	CompanionNames  []string  `json:"companion_names"`
	Message         string    `json:"message,omitempty"`
	RespondedAt     time.Time `json:"responded_at"`
}

func NewRSVPResponse(rsvp *models.RSVP, guest *models.Guest) RSVPResponse {
	names := rsvp.CompanionNames
	if names == nil {
		names = []string{}
	}
	return RSVPResponse{
		ID:              rsvp.ID,
		EventID:         rsvp.EventID,
		GuestID:         rsvp.GuestID,
		GuestName:       guest.Name,
		GuestShortCode:  guest.ShortCode,
		QRCodeToken:     guest.QRCodeToken,
		Status:          rsvp.Status,
		CompanionsCount: rsvp.CompanionsCount,
		CompanionNames:  names,
		Message:         rsvp.Message,
		RespondedAt:     rsvp.RespondedAt,
	}
}
