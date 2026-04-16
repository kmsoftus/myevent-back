package dto

import "myevent-back/internal/models"

type CreateEventRequest struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Type            string `json:"type"`
	Description     string `json:"description"`
	Date            string `json:"date"`
	Time            string `json:"time"`
	LocationName    string `json:"location_name"`
	Address         string `json:"address"`
	CoverImageURL   string `json:"cover_image_url"`
	HostMessage     string `json:"host_message"`
	Theme           string `json:"theme"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	PixKey          string `json:"pix_key"`
	PixHolderName   string `json:"pix_holder_name"`
	PixBank         string `json:"pix_bank"`
	OpenRSVP        bool   `json:"open_rsvp"`
}

type UpdateEventRequest struct {
	Title           *string `json:"title"`
	Slug            *string `json:"slug"`
	Type            *string `json:"type"`
	Description     *string `json:"description"`
	Date            *string `json:"date"`
	Time            *string `json:"time"`
	LocationName    *string `json:"location_name"`
	Address         *string `json:"address"`
	CoverImageURL   *string `json:"cover_image_url"`
	HostMessage     *string `json:"host_message"`
	Theme           *string `json:"theme"`
	PrimaryColor    *string `json:"primary_color"`
	SecondaryColor  *string `json:"secondary_color"`
	BackgroundColor *string `json:"background_color"`
	TextColor       *string `json:"text_color"`
	PixKey          *string `json:"pix_key"`
	PixHolderName   *string `json:"pix_holder_name"`
	PixBank         *string `json:"pix_bank"`
	OpenRSVP        *bool   `json:"open_rsvp"`
}

type UpdateEventStatusRequest struct {
	Status string `json:"status"`
}

type ListEventsRequest struct {
	Query  string
	Status string
	Sort   string
}

type EventResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Type            string `json:"type"`
	Description     string `json:"description"`
	Date            string `json:"date"`
	Time            string `json:"time"`
	LocationName    string `json:"location_name"`
	Address         string `json:"address"`
	CoverImageURL   string `json:"cover_image_url,omitempty"`
	HostMessage     string `json:"host_message"`
	Theme           string `json:"theme"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	PixKey          string `json:"pix_key,omitempty"`
	PixHolderName   string `json:"pix_holder_name,omitempty"`
	PixBank         string `json:"pix_bank,omitempty"`
	Status          string `json:"status"`
	OpenRSVP        bool   `json:"open_rsvp"`
}

type PublicEventResponse struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Type            string `json:"type"`
	Description     string `json:"description"`
	Date            string `json:"date"`
	Time            string `json:"time"`
	LocationName    string `json:"location_name"`
	Address         string `json:"address"`
	CoverImageURL   string `json:"cover_image_url,omitempty"`
	HostMessage     string `json:"host_message"`
	Theme           string `json:"theme"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	PixKey          string `json:"pix_key,omitempty"`
	PixHolderName   string `json:"pix_holder_name,omitempty"`
	PixBank         string `json:"pix_bank,omitempty"`
	Status          string `json:"status"`
	OpenRSVP        bool   `json:"open_rsvp"`
}

func NewEventResponse(event *models.Event) EventResponse {
	return EventResponse{
		ID:              event.ID,
		UserID:          event.UserID,
		Title:           event.Title,
		Slug:            event.Slug,
		Type:            event.Type,
		Description:     event.Description,
		Date:            event.Date,
		Time:            event.Time,
		LocationName:    event.LocationName,
		Address:         event.Address,
		CoverImageURL:   event.CoverImageURL,
		HostMessage:     event.HostMessage,
		Theme:           event.Theme,
		PrimaryColor:    event.PrimaryColor,
		SecondaryColor:  event.SecondaryColor,
		BackgroundColor: event.BackgroundColor,
		TextColor:       event.TextColor,
		PixKey:          event.PixKey,
		PixHolderName:   event.PixHolderName,
		PixBank:         event.PixBank,
		Status:          event.Status,
		OpenRSVP:        event.OpenRSVP,
	}
}

func NewPublicEventResponse(event *models.Event) PublicEventResponse {
	return PublicEventResponse{
		ID:              event.ID,
		Title:           event.Title,
		Slug:            event.Slug,
		Type:            event.Type,
		Description:     event.Description,
		Date:            event.Date,
		Time:            event.Time,
		LocationName:    event.LocationName,
		Address:         event.Address,
		CoverImageURL:   event.CoverImageURL,
		HostMessage:     event.HostMessage,
		Theme:           event.Theme,
		PrimaryColor:    event.PrimaryColor,
		SecondaryColor:  event.SecondaryColor,
		BackgroundColor: event.BackgroundColor,
		TextColor:       event.TextColor,
		PixKey:          event.PixKey,
		PixHolderName:   event.PixHolderName,
		PixBank:         event.PixBank,
		Status:          event.Status,
		OpenRSVP:        event.OpenRSVP,
	}
}
