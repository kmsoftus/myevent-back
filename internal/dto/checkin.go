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
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	RSVPStatus  string     `json:"rsvp_status"`
	CheckedInAt *time.Time `json:"checked_in_at,omitempty"`
}

func NewCheckInResponse(guest *models.Guest) CheckInResponse {
	return CheckInResponse{
		Success: true,
		Guest: CheckInGuestPayload{
			ID:          guest.ID,
			Name:        guest.Name,
			RSVPStatus:  guest.RSVPStatus,
			CheckedInAt: guest.CheckedInAt,
		},
	}
}
