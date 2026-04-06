package models

import "time"

type GalleryPhoto struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	ImageURL  string    `json:"image_url"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}
