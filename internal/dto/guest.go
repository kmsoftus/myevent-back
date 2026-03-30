package dto

import (
	"time"

	"myevent-back/internal/models"
)

type CreateGuestRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	MaxCompanions int    `json:"max_companions"`
}

type UpdateGuestRequest struct {
	Name          *string `json:"name"`
	Email         *string `json:"email"`
	Phone         *string `json:"phone"`
	MaxCompanions *int    `json:"max_companions"`
}

type GuestResponse struct {
	ID            string     `json:"id"`
	EventID       string     `json:"event_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	InviteCode    string     `json:"invite_code"`
	QRCodeToken   string     `json:"qr_code_token"`
	MaxCompanions int        `json:"max_companions"`
	RSVPStatus    string     `json:"rsvp_status"`
	CheckedInAt   *time.Time `json:"checked_in_at,omitempty"`
}

func NewGuestResponse(guest *models.Guest) GuestResponse {
	return GuestResponse{
		ID:            guest.ID,
		EventID:       guest.EventID,
		Name:          guest.Name,
		Email:         guest.Email,
		Phone:         guest.Phone,
		InviteCode:    guest.InviteCode,
		QRCodeToken:   guest.QRCodeToken,
		MaxCompanions: guest.MaxCompanions,
		RSVPStatus:    guest.RSVPStatus,
		CheckedInAt:   guest.CheckedInAt,
	}
}
