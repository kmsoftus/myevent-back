package dto

import "myevent-back/internal/models"

type GuestQRCodeResponse struct {
	GuestID     string `json:"guest_id"`
	QRCodeToken string `json:"qr_code_token"`
	CheckinURL  string `json:"checkin_url"`
}

func NewGuestQRCodeResponse(guest *models.Guest) GuestQRCodeResponse {
	return GuestQRCodeResponse{
		GuestID:     guest.ID,
		QRCodeToken: guest.QRCodeToken,
		CheckinURL:  "/checkin/" + guest.QRCodeToken,
	}
}
