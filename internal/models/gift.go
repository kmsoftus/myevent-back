package models

import "time"

type Gift struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description,omitempty"`
	ImageURL         string    `json:"image_url,omitempty"`
	ValueCents       *int      `json:"value_cents,omitempty"`
	ExternalLink     string    `json:"external_link,omitempty"`
	Status           string    `json:"status"`
	AllowReservation bool      `json:"allow_reservation"`
	AllowPix         bool      `json:"allow_pix"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
