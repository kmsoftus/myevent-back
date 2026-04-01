package dto

import (
	"time"

	"myevent-back/internal/models"
)

type CheckInRequest struct {
	QRCodeToken string `json:"qr_code_token"`
	GuestID     string `json:"guest_id"`
}

type CheckInResponse struct {
	Success bool                `json:"success"`
	Guest   CheckInGuestPayload `json:"guest"`
}

type CheckInGuestPayload struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	ShortCode      string     `json:"short_code"`
	RSVPStatus     string     `json:"rsvp_status"`
	MaxCompanions  int        `json:"max_companions"`
	CheckedInAt    *time.Time `json:"checked_in_at,omitempty"`
	AlreadyCheckedIn bool     `json:"already_checked_in"`
}

func NewCheckInResponse(guest *models.Guest, alreadyCheckedIn bool) CheckInResponse {
	return CheckInResponse{
		Success: true,
		Guest: CheckInGuestPayload{
			ID:               guest.ID,
			Name:             guest.Name,
			ShortCode:        guest.ShortCode,
			RSVPStatus:       guest.RSVPStatus,
			MaxCompanions:    guest.MaxCompanions,
			CheckedInAt:      guest.CheckedInAt,
			AlreadyCheckedIn: alreadyCheckedIn,
		},
	}
}
