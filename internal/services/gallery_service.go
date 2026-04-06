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

const maxGalleryPhotos = 10

type GalleryService struct {
	events  repositories.EventRepository
	gallery repositories.GalleryPhotoRepository
}

func NewGalleryService(events repositories.EventRepository, gallery repositories.GalleryPhotoRepository) *GalleryService {
	return &GalleryService{events: events, gallery: gallery}
}

func (s *GalleryService) AddPhoto(ctx context.Context, userID, eventID string, input dto.AddGalleryPhotoRequest) (*models.GalleryPhoto, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	imageURL := strings.TrimSpace(input.ImageURL)
	if imageURL == "" {
		return nil, fmt.Errorf("%w: Informe a URL da foto.", ErrValidation)
	}
	if err := validateOptionalURL("image_url", imageURL); err != nil {
		return nil, err
	}

	count, err := s.gallery.CountByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if count >= maxGalleryPhotos {
		return nil, fmt.Errorf("%w: O limite de %d fotos na galeria foi atingido.", ErrValidation, maxGalleryPhotos)
	}

	photo := &models.GalleryPhoto{
		ID:        uuid.NewString(),
		EventID:   eventID,
		ImageURL:  imageURL,
		Position:  count,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.gallery.Create(ctx, photo); err != nil {
		return nil, err
	}

	return photo, nil
}

func (s *GalleryService) ListByEvent(ctx context.Context, userID, eventID string) ([]*models.GalleryPhoto, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	return s.gallery.ListByEventID(ctx, eventID)
}

func (s *GalleryService) ListPublicBySlug(ctx context.Context, slug string) ([]*models.GalleryPhoto, error) {
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

	return s.gallery.ListByEventID(ctx, event.ID)
}

func (s *GalleryService) DeletePhoto(ctx context.Context, userID, eventID, photoID string) error {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return err
	}

	photo, err := s.gallery.GetByID(ctx, photoID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if photo.EventID != eventID {
		return ErrNotFound
	}

	return s.gallery.Delete(ctx, photo.ID)
}

func (s *GalleryService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
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
