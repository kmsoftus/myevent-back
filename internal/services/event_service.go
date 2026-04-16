package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
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
		input.Description,
		input.Date,
		"",
		input.Time,
		input.HostMessage,
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
		PixBank:         strings.TrimSpace(input.PixBank),
		Status:          "published",
		OpenRSVP:        input.OpenRSVP,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.events.Create(ctx, event); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: Este endereco ja esta sendo usado por outro evento. Tente um diferente.", ErrConflict)
		}
		return nil, err
	}

	return event, nil
}

func (s *EventService) ListByUser(ctx context.Context, userID string, input dto.ListEventsRequest) ([]*models.Event, error) {
	statusFilter := strings.TrimSpace(strings.ToLower(input.Status))
	if statusFilter != "" && !isValidEventStatus(statusFilter) {
		return nil, fmt.Errorf("%w: Status do evento invalido.", ErrValidation)
	}

	sortOption := normalizeListEventsSort(input.Sort)
	if sortOption == "" {
		return nil, fmt.Errorf("%w: Ordenacao invalida.", ErrValidation)
	}

	events, err := s.events.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	queryFilter := strings.ToLower(strings.TrimSpace(input.Query))
	filtered := make([]*models.Event, 0, len(events))
	for _, event := range events {
		if statusFilter != "" && event.Status != statusFilter {
			continue
		}
		if queryFilter != "" && !strings.Contains(strings.ToLower(event.Title), queryFilter) {
			continue
		}
		filtered = append(filtered, event)
	}

	sortEventsByDate(filtered, sortOption)
	return filtered, nil
}

func (s *EventService) GetByIDForUser(ctx context.Context, userID, eventID string) (*models.Event, error) {
	if _, err := uuid.Parse(eventID); err != nil {
		return nil, ErrNotFound
	}

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
	nextDescription := coalesceString(input.Description, event.Description)
	nextHostMessage := coalesceString(input.HostMessage, event.HostMessage)
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
		nextDescription,
		nextDate,
		event.Date,
		nextTime,
		nextHostMessage,
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
	event.Description = strings.TrimSpace(nextDescription)
	event.Date = strings.TrimSpace(nextDate)
	event.Time = strings.TrimSpace(nextTime)
	event.LocationName = strings.TrimSpace(coalesceString(input.LocationName, event.LocationName))
	event.Address = strings.TrimSpace(coalesceString(input.Address, event.Address))
	event.CoverImageURL = strings.TrimSpace(nextCoverImageURL)
	event.HostMessage = strings.TrimSpace(nextHostMessage)
	event.Theme = nextTheme
	event.PrimaryColor = strings.TrimSpace(nextPrimaryColor)
	event.SecondaryColor = strings.TrimSpace(nextSecondaryColor)
	event.BackgroundColor = strings.TrimSpace(nextBackgroundColor)
	event.TextColor = strings.TrimSpace(nextTextColor)
	event.PixKey = strings.TrimSpace(coalesceString(input.PixKey, event.PixKey))
	event.PixHolderName = strings.TrimSpace(coalesceString(input.PixHolderName, event.PixHolderName))
	event.PixBank = strings.TrimSpace(coalesceString(input.PixBank, event.PixBank))
	if input.OpenRSVP != nil {
		event.OpenRSVP = *input.OpenRSVP
	}
	event.UpdatedAt = time.Now().UTC()

	if err := s.events.Update(ctx, event); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("%w: Este endereco ja esta sendo usado por outro evento. Tente um diferente.", ErrConflict)
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
		return nil, fmt.Errorf("%w: Status do evento invalido.", ErrValidation)
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

func (s *EventService) validateAndNormalizePayload(title, slug, description, date, previousDate, hour, hostMessage, theme, primaryColor, secondaryColor, backgroundColor, textColor, coverImageURL string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("%w: Informe o nome do evento.", ErrValidation)
	}

	normalizedSlug := utils.NormalizeSlug(slug)
	if normalizedSlug == "" {
		normalizedSlug = utils.NormalizeSlug(title)
	}
	if normalizedSlug == "" {
		return "", fmt.Errorf("%w: Informe um endereco valido para o evento.", ErrValidation)
	}

	date = strings.TrimSpace(date)
	if date != "" {
		parsedDate, err := time.ParseInLocation("2006-01-02", date, saoPauloLocation)
		if err != nil {
			return "", fmt.Errorf("%w: A data deve estar no formato AAAA-MM-DD.", ErrValidation)
		}

		today := startOfDayInLocation(time.Now(), saoPauloLocation)
		if parsedDate.Before(today) && strings.TrimSpace(previousDate) != date {
			return "", fmt.Errorf("%w: A data do evento nao pode ser anterior a hoje.", ErrValidation)
		}
	}

	hour = strings.TrimSpace(hour)
	if hour != "" {
		if _, err := time.Parse("15:04", hour); err != nil {
			return "", fmt.Errorf("%w: O horario deve estar no formato HH:MM.", ErrValidation)
		}
	}

	if err := validateTheme(theme); err != nil {
		return "", err
	}
	if err := validateTextMaxLength("description", "descricao", description, maxEventDescriptionLength, "event_description_too_long"); err != nil {
		return "", err
	}
	if err := validateTextMaxLength("host_message", "mensagem dos anfitrioes", hostMessage, maxEventHostMessageLength, "event_host_message_too_long"); err != nil {
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

func normalizeListEventsSort(sortInput string) string {
	switch strings.TrimSpace(strings.ToLower(sortInput)) {
	case "", "date_desc":
		return "date_desc"
	case "date_asc":
		return "date_asc"
	default:
		return ""
	}
}

func sortEventsByDate(events []*models.Event, sortOption string) {
	sort.SliceStable(events, func(i, j int) bool {
		leftDate, leftValid := parseEventDate(events[i].Date)
		rightDate, rightValid := parseEventDate(events[j].Date)

		if leftValid && rightValid && !leftDate.Equal(rightDate) {
			if sortOption == "date_asc" {
				return leftDate.Before(rightDate)
			}
			return leftDate.After(rightDate)
		}
		if leftValid != rightValid {
			return leftValid
		}

		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
}

func parseEventDate(rawDate string) (time.Time, bool) {
	parsedDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(rawDate), saoPauloLocation)
	if err != nil {
		return time.Time{}, false
	}
	return parsedDate, true
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
		return fmt.Errorf("%w: Tema invalido.", ErrValidation)
	}
}

func validateHexColor(field, color string) error {
	color = strings.TrimSpace(color)
	if color == "" {
		return nil
	}
	if !hexColorPattern.MatchString(color) {
		return fmt.Errorf("%w: A cor do campo %s deve estar no formato #RRGGBB.", ErrValidation, field)
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
