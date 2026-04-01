package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type GiftService struct {
	events repositories.EventRepository
	gifts  repositories.GiftRepository
}

func NewGiftService(events repositories.EventRepository, gifts repositories.GiftRepository) *GiftService {
	return &GiftService{
		events: events,
		gifts:  gifts,
	}
}

func (s *GiftService) Create(ctx context.Context, userID, eventID string, input dto.CreateGiftRequest) (*models.Gift, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}
	if err := validateGiftPayload(input.Title, input.ValueCents); err != nil {
		return nil, err
	}
	if err := validateGiftLinks(input.ImageURL, input.ExternalLink); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	gift := &models.Gift{
		ID:               uuid.NewString(),
		EventID:          eventID,
		Title:            strings.TrimSpace(input.Title),
		Description:      strings.TrimSpace(input.Description),
		ImageURL:         strings.TrimSpace(input.ImageURL),
		ValueCents:       copyIntPointer(input.ValueCents),
		ExternalLink:     strings.TrimSpace(input.ExternalLink),
		Status:           "available",
		AllowReservation: coalesceBool(input.AllowReservation, true),
		AllowPix:         coalesceBool(input.AllowPix, true),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.gifts.Create(ctx, gift); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return gift, nil
}

func (s *GiftService) ListByEvent(ctx context.Context, userID, eventID string) ([]*models.Gift, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	return s.gifts.ListByEventID(ctx, eventID)
}

type PagedGifts struct {
	Gifts      []*models.Gift
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

func (s *GiftService) ListPublicBySlug(ctx context.Context, slug string, page, pageSize int) (*PagedGifts, error) {
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

	total, err := s.gifts.CountByEventID(ctx, event.ID)
	if err != nil {
		return nil, err
	}

	// When no pagination params are given, return all gifts
	if page < 1 && pageSize < 1 {
		gifts, err := s.gifts.ListByEventID(ctx, event.ID)
		if err != nil {
			return nil, err
		}
		return &PagedGifts{
			Gifts:      gifts,
			Total:      total,
			Page:       1,
			PageSize:   total,
			TotalPages: 1,
		}, nil
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 12
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	offset := (page - 1) * pageSize
	gifts, err := s.gifts.ListByEventIDPaged(ctx, event.ID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &PagedGifts{
		Gifts:      gifts,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *GiftService) GetByID(ctx context.Context, userID, eventID, giftID string) (*models.Gift, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	gift, err := s.gifts.GetByID(ctx, giftID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if gift.EventID != eventID {
		return nil, ErrNotFound
	}

	return gift, nil
}

func (s *GiftService) Update(ctx context.Context, userID, eventID, giftID string, input dto.UpdateGiftRequest) (*models.Gift, error) {
	gift, err := s.GetByID(ctx, userID, eventID, giftID)
	if err != nil {
		return nil, err
	}

	nextTitle := coalesceString(input.Title, gift.Title)
	nextValueCents := coalesceOptionalInt(input.ValueCents, gift.ValueCents)
	if err := validateGiftPayload(nextTitle, nextValueCents); err != nil {
		return nil, err
	}
	nextImageURL := coalesceString(input.ImageURL, gift.ImageURL)
	nextExternalLink := coalesceString(input.ExternalLink, gift.ExternalLink)
	if err := validateGiftLinks(nextImageURL, nextExternalLink); err != nil {
		return nil, err
	}

	gift.Title = strings.TrimSpace(nextTitle)
	gift.Description = strings.TrimSpace(coalesceString(input.Description, gift.Description))
	gift.ImageURL = strings.TrimSpace(nextImageURL)
	gift.ValueCents = copyIntPointer(nextValueCents)
	gift.ExternalLink = strings.TrimSpace(nextExternalLink)
	gift.AllowReservation = coalesceOptionalBool(input.AllowReservation, gift.AllowReservation)
	gift.AllowPix = coalesceOptionalBool(input.AllowPix, gift.AllowPix)
	gift.UpdatedAt = time.Now().UTC()

	if err := s.gifts.Update(ctx, gift); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return gift, nil
}

func (s *GiftService) Delete(ctx context.Context, userID, eventID, giftID string) error {
	gift, err := s.GetByID(ctx, userID, eventID, giftID)
	if err != nil {
		return err
	}

	if err := s.gifts.Delete(ctx, gift.ID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (s *GiftService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
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

func validateGiftPayload(title string, valueCents *int) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("%w: Informe o nome do presente.", ErrValidation)
	}
	if valueCents != nil && *valueCents < 0 {
		return fmt.Errorf("%w: O valor do presente nao pode ser negativo.", ErrValidation)
	}
	return nil
}

func validateGiftLinks(imageURL, externalLink string) error {
	if err := validateOptionalURL("image_url", imageURL); err != nil {
		return err
	}
	if err := validateOptionalURL("external_link", externalLink); err != nil {
		return err
	}
	return nil
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func coalesceBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func coalesceOptionalBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func coalesceOptionalInt(value *int, fallback *int) *int {
	if value == nil {
		return copyIntPointer(fallback)
	}
	return copyIntPointer(value)
}
