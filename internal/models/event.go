package models

import "time"

type Event struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	Type            string    `json:"type"`
	Description     string    `json:"description"`
	Date            string    `json:"date"`
	Time            string    `json:"time"`
	LocationName    string    `json:"location_name"`
	Address         string    `json:"address"`
	CoverImageURL   string    `json:"cover_image_url"`
	HostMessage     string    `json:"host_message"`
	Theme           string    `json:"theme"`
	PrimaryColor    string    `json:"primary_color"`
	SecondaryColor  string    `json:"secondary_color"`
	BackgroundColor string    `json:"background_color"`
	TextColor       string    `json:"text_color"`
	PixKey          string    `json:"pix_key"`
	PixHolderName   string    `json:"pix_holder_name"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
