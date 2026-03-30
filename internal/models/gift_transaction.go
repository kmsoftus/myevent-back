package models

import "time"

type GiftTransaction struct {
	ID           string     `json:"id"`
	GiftID       string     `json:"gift_id"`
	EventID      string     `json:"event_id"`
	GuestName    string     `json:"guest_name"`
	GuestContact string     `json:"guest_contact,omitempty"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	Message      string     `json:"message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
