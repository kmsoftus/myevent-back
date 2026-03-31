package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
	"myevent-back/internal/utils"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var saoPauloLocation = time.FixedZone("America/Sao_Paulo", -3*60*60)

type EventService struct {
	events repositories.EventRepository
}

func NewEventService(events repositories.EventRepository) *EventService {
	return &EventService{events: events}
}

func (s *EventService) Create(ctx context.Context, userID string, input dto.CreateEventRequest) (*models.Event, error) {
	theme := normalizeTheme(input.Theme)
	slug, err := s.validateAndNormalizePayload(
		input.Title,
		input.Slug,
		input.Date,
		"",
		input.Time,
		theme,
		input.PrimaryColor,
		input.SecondaryColor,
		input.BackgroundColor,
		input.TextColor,
		input.CoverImageURL,
	)
	if err != nil {
		return nil, err
	}

	palette := paletteForTheme(theme)
	now := time.Now().UTC()
	event := &models.Event{
		ID:              uuid.NewString(),
		UserID:          userID,
		Title:           strings.TrimSpace(input.Title),
		Slug:            slug,
		Type:            strings.TrimSpace(input.Type),
		Description:     strings.TrimSpace(input.Description),
		Date:            strings.TrimSpace(input.Date),
		Time:            strings.TrimSpace(input.Time),
		LocationName:    strings.TrimSpace(input.LocationName),
		Address:         strings.TrimSpace(input.Address),
		CoverImageURL:   strings.TrimSpace(input.CoverImageURL),
		HostMessage:     strings.TrimSpace(input.HostMessage),
		Theme:           theme,
		PrimaryColor:    defaultString(strings.TrimSpace(input.PrimaryColor), palette.PrimaryColor),
		SecondaryColor:  defaultString(strings.TrimSpace(input.SecondaryColor), palette.SecondaryColor),
		BackgroundColor: defaultString(strings.TrimSpace(input.BackgroundColor), palette.BackgroundColor),
		TextColor:       defaultString(strings.TrimSpace(input.TextColor), palette.TextColor),
		PixKey:          strings.TrimSpace(input.PixKey),
		PixHolderName:   strings.TrimSpace(input.PixHolderName),
		Status:          "draft",
		OpenRSVP:        input.OpenRSVP,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.events.Create(ctx, event); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: slug already in use", ErrConflict)
		}
		return nil, err
	}

	return event, nil
}

func (s *EventService) ListByUser(ctx context.Context, userID string) ([]*models.Event, error) {
	return s.events.ListByUserID(ctx, userID)
}

func (s *EventService) GetByIDForUser(ctx context.Context, userID, eventID string) (*models.Event, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if event.UserID != userID {
		return nil, ErrForbidden
	}

	return event, nil
}

func (s *EventService) Update(ctx context.Context, userID, eventID string, input dto.UpdateEventRequest) (*models.Event, error) {
	event, err := s.GetByIDForUser(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}

	nextTitle := coalesceString(input.Title, event.Title)
	nextSlug := coalesceString(input.Slug, event.Slug)
	nextDate := coalesceString(input.Date, event.Date)
	nextTime := coalesceString(input.Time, event.Time)
	nextTheme := normalizeTheme(coalesceString(input.Theme, event.Theme))
	nextPrimaryColor := resolveEventColorUpdate(input.PrimaryColor, event.PrimaryColor, event.Theme, nextTheme, func(palette themePalette) string {
		return palette.PrimaryColor
	})
	nextSecondaryColor := resolveEventColorUpdate(input.SecondaryColor, event.SecondaryColor, event.Theme, nextTheme, func(palette themePalette) string {
		return palette.SecondaryColor
	})
	nextBackgroundColor := resolveEventColorUpdate(input.BackgroundColor, event.BackgroundColor, event.Theme, nextTheme, func(palette themePalette) string {
		return palette.BackgroundColor
	})
	nextTextColor := resolveEventColorUpdate(input.TextColor, event.TextColor, event.Theme, nextTheme, func(palette themePalette) string {
		return palette.TextColor
	})
	nextCoverImageURL := coalesceString(input.CoverImageURL, event.CoverImageURL)

	slug, err := s.validateAndNormalizePayload(
		nextTitle,
		nextSlug,
		nextDate,
		event.Date,
		nextTime,
		nextTheme,
		nextPrimaryColor,
		nextSecondaryColor,
		nextBackgroundColor,
		nextTextColor,
		nextCoverImageURL,
	)
	if err != nil {
		return nil, err
	}

	event.Title = strings.TrimSpace(nextTitle)
	event.Slug = slug
	event.Type = strings.TrimSpace(coalesceString(input.Type, event.Type))
	event.Description = strings.TrimSpace(coalesceString(input.Description, event.Description))
	event.Date = strings.TrimSpace(nextDate)
	event.Time = strings.TrimSpace(nextTime)
	event.LocationName = strings.TrimSpace(coalesceString(input.LocationName, event.LocationName))
	event.Address = strings.TrimSpace(coalesceString(input.Address, event.Address))
	event.CoverImageURL = strings.TrimSpace(nextCoverImageURL)
	event.HostMessage = strings.TrimSpace(coalesceString(input.HostMessage, event.HostMessage))
	event.Theme = nextTheme
	event.PrimaryColor = strings.TrimSpace(nextPrimaryColor)
	event.SecondaryColor = strings.TrimSpace(nextSecondaryColor)
	event.BackgroundColor = strings.TrimSpace(nextBackgroundColor)
	event.TextColor = strings.TrimSpace(nextTextColor)
	event.PixKey = strings.TrimSpace(coalesceString(input.PixKey, event.PixKey))
	event.PixHolderName = strings.TrimSpace(coalesceString(input.PixHolderName, event.PixHolderName))
	if input.OpenRSVP != nil {
		event.OpenRSVP = *input.OpenRSVP
	}
	event.UpdatedAt = time.Now().UTC()

	if err := s.events.Update(ctx, event); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: slug already in use", ErrConflict)
		}
		return nil, err
	}

	return event, nil
}

func (s *EventService) UpdateStatus(ctx context.Context, userID, eventID, status string) (*models.Event, error) {
	event, err := s.GetByIDForUser(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}

	status = strings.TrimSpace(strings.ToLower(status))
	if !isValidEventStatus(status) {
		return nil, fmt.Errorf("%w: invalid event status", ErrValidation)
	}

	event.Status = status
	event.UpdatedAt = time.Now().UTC()

	if err := s.events.Update(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *EventService) Delete(ctx context.Context, userID, eventID string) error {
	event, err := s.GetByIDForUser(ctx, userID, eventID)
	if err != nil {
		return err
	}

	if err := s.events.Delete(ctx, event.ID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (s *EventService) GetPublishedBySlug(ctx context.Context, slug string) (*models.Event, error) {
	event, err := s.events.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if event.Status != "published" {
		return nil, ErrNotFound
	}

	return event, nil
}

func (s *EventService) validateAndNormalizePayload(title, slug, date, previousDate, hour, theme, primaryColor, secondaryColor, backgroundColor, textColor, coverImageURL string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("%w: title is required", ErrValidation)
	}

	normalizedSlug := utils.NormalizeSlug(slug)
	if normalizedSlug == "" {
		normalizedSlug = utils.NormalizeSlug(title)
	}
	if normalizedSlug == "" {
		return "", fmt.Errorf("%w: invalid slug", ErrValidation)
	}

	date = strings.TrimSpace(date)
	if date != "" {
		parsedDate, err := time.ParseInLocation("2006-01-02", date, saoPauloLocation)
		if err != nil {
			return "", fmt.Errorf("%w: date must use YYYY-MM-DD", ErrValidation)
		}

		today := startOfDayInLocation(time.Now(), saoPauloLocation)
		if parsedDate.Before(today) && strings.TrimSpace(previousDate) != date {
			return "", fmt.Errorf("%w: A data do evento nao pode ser anterior a hoje.", ErrValidation)
		}
	}

	hour = strings.TrimSpace(hour)
	if hour != "" {
		if _, err := time.Parse("15:04", hour); err != nil {
			return "", fmt.Errorf("%w: time must use HH:MM", ErrValidation)
		}
	}

	if err := validateTheme(theme); err != nil {
		return "", err
	}
	if err := validateOptionalURL("cover_image_url", coverImageURL); err != nil {
		return "", err
	}
	if err := validateHexColor("primary_color", primaryColor); err != nil {
		return "", err
	}
	if err := validateHexColor("secondary_color", secondaryColor); err != nil {
		return "", err
	}
	if err := validateHexColor("background_color", backgroundColor); err != nil {
		return "", err
	}
	if err := validateHexColor("text_color", textColor); err != nil {
		return "", err
	}

	return normalizedSlug, nil
}

func validateTheme(theme string) error {
	theme = strings.TrimSpace(strings.ToLower(theme))
	if theme == "" {
		return nil
	}

	switch theme {
	case "classic", "minimal", "party":
		return nil
	default:
		return fmt.Errorf("%w: invalid theme", ErrValidation)
	}
}

func validateHexColor(field, color string) error {
	color = strings.TrimSpace(color)
	if color == "" {
		return nil
	}
	if !hexColorPattern.MatchString(color) {
		return fmt.Errorf("%w: %s must use #RRGGBB", ErrValidation, field)
	}
	return nil
}

func isValidEventStatus(status string) bool {
	switch status {
	case "draft", "published", "closed":
		return true
	default:
		return false
	}
}

func resolveEventColorUpdate(input *string, current, currentTheme, nextTheme string, paletteValue func(themePalette) string) string {
	if input != nil {
		return strings.TrimSpace(*input)
	}

	currentTheme = normalizeTheme(currentTheme)
	nextTheme = normalizeTheme(nextTheme)
	if currentTheme == nextTheme {
		return current
	}

	currentPalette := paletteForTheme(currentTheme)
	if current == paletteValue(currentPalette) {
		return paletteValue(paletteForTheme(nextTheme))
	}

	return current
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func startOfDayInLocation(now time.Time, location *time.Location) time.Time {
	current := now.In(location)
	return time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, location)
}

func coalesceString(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
