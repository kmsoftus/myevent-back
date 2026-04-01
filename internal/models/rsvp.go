package models

import "time"

type RSVP struct {
	ID              string    `json:"id"`
	EventID         string    `json:"event_id"`
	GuestID         string    `json:"guest_id"`
	Status          string    `json:"status"`
	CompanionsCount int       `json:"companions_count"`
	CompanionNames  []string  `json:"companion_names"`
	Message         string    `json:"message,omitempty"`
	RespondedAt     time.Time `json:"responded_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
