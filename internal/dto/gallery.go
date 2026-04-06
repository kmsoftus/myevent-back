package dto

import "myevent-back/internal/models"

type AddGalleryPhotoRequest struct {
	ImageURL string `json:"image_url"`
}

type GalleryPhotoResponse struct {
	ID        string `json:"id"`
	EventID   string `json:"event_id"`
	ImageURL  string `json:"image_url"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
}

func NewGalleryPhotoResponse(p *models.GalleryPhoto) GalleryPhotoResponse {
	return GalleryPhotoResponse{
		ID:        p.ID,
		EventID:   p.EventID,
		ImageURL:  p.ImageURL,
		Position:  p.Position,
		CreatedAt: p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
