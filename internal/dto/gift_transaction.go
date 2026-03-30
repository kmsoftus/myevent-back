package dto

import (
	"time"

	"myevent-back/internal/models"
)

type CreateGiftTransactionRequest struct {
	GuestName    string `json:"guest_name"`
	GuestContact string `json:"guest_contact"`
	Message      string `json:"message"`
}

type UpdateGiftTransactionStatusRequest struct {
	Status string `json:"status"`
}

type GiftTransactionResponse struct {
	ID           string     `json:"id"`
	GiftID       string     `json:"gift_id"`
	GiftTitle    string     `json:"gift_title"`
	EventID      string     `json:"event_id"`
	GuestName    string     `json:"guest_name"`
	GuestContact string     `json:"guest_contact,omitempty"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	Message      string     `json:"message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`
}

func NewGiftTransactionResponse(transaction *models.GiftTransaction, gift *models.Gift) GiftTransactionResponse {
	return GiftTransactionResponse{
		ID:           transaction.ID,
		GiftID:       transaction.GiftID,
		GiftTitle:    gift.Title,
		EventID:      transaction.EventID,
		GuestName:    transaction.GuestName,
		GuestContact: transaction.GuestContact,
		Type:         transaction.Type,
		Status:       transaction.Status,
		Message:      transaction.Message,
		CreatedAt:    transaction.CreatedAt,
		ConfirmedAt:  transaction.ConfirmedAt,
	}
}
