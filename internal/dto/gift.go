package dto

import "myevent-back/internal/models"

type CreateGiftRequest struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	ImageURL         string `json:"image_url"`
	ValueCents       *int   `json:"value_cents"`
	ExternalLink     string `json:"external_link"`
	AllowReservation *bool  `json:"allow_reservation"`
	AllowPix         *bool  `json:"allow_pix"`
}

type UpdateGiftRequest struct {
	Title            *string `json:"title"`
	Description      *string `json:"description"`
	ImageURL         *string `json:"image_url"`
	ValueCents       *int    `json:"value_cents"`
	ExternalLink     *string `json:"external_link"`
	AllowReservation *bool   `json:"allow_reservation"`
	AllowPix         *bool   `json:"allow_pix"`
}

type GiftResponse struct {
	ID               string `json:"id"`
	EventID          string `json:"event_id"`
	Title            string `json:"title"`
	Description      string `json:"description,omitempty"`
	ImageURL         string `json:"image_url,omitempty"`
	ValueCents       *int   `json:"value_cents,omitempty"`
	ExternalLink     string `json:"external_link,omitempty"`
	Status           string `json:"status"`
	AllowReservation bool   `json:"allow_reservation"`
	AllowPix         bool   `json:"allow_pix"`
}

type PagedGiftsResponse struct {
	Items      []GiftResponse `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

func NewGiftResponse(gift *models.Gift) GiftResponse {
	return GiftResponse{
		ID:               gift.ID,
		EventID:          gift.EventID,
		Title:            gift.Title,
		Description:      gift.Description,
		ImageURL:         gift.ImageURL,
		ValueCents:       gift.ValueCents,
		ExternalLink:     gift.ExternalLink,
		Status:           gift.Status,
		AllowReservation: gift.AllowReservation,
		AllowPix:         gift.AllowPix,
	}
}
