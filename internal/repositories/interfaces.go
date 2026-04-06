package repositories

import (
	"context"
	"errors"
	"time"

	"myevent-back/internal/models"
)

var (
	ErrNotFound = errors.New("repository not found")
	ErrConflict = errors.New("repository conflict")
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Delete(ctx context.Context, id string) error
	UpdateProfile(ctx context.Context, id, name, contactPhone string, updatedAt time.Time) error
	UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error
}

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *models.PasswordResetToken) error
	DeleteActiveByUserID(ctx context.Context, userID string, now time.Time) error
	Consume(ctx context.Context, tokenHash string, now time.Time) (*models.PasswordResetToken, error)
}

type EventRepository interface {
	Create(ctx context.Context, event *models.Event) error
	ListByUserID(ctx context.Context, userID string) ([]*models.Event, error)
	GetByID(ctx context.Context, id string) (*models.Event, error)
	GetBySlug(ctx context.Context, slug string) (*models.Event, error)
	Update(ctx context.Context, event *models.Event) error
	Delete(ctx context.Context, id string) error
}

type GuestRepository interface {
	Create(ctx context.Context, guest *models.Guest) error
	ListByEventID(ctx context.Context, eventID string) ([]*models.Guest, error)
	CountByEventID(ctx context.Context, eventID string) (int, error)
	ListByEventIDPaged(ctx context.Context, eventID string, limit, offset int) ([]*models.Guest, error)
	GetByID(ctx context.Context, id string) (*models.Guest, error)
	GetByInviteCode(ctx context.Context, inviteCode string) (*models.Guest, error)
	GetByShortCode(ctx context.Context, eventID, shortCode string) (*models.Guest, error)
	SearchByName(ctx context.Context, eventID, query string, limit int) ([]*models.Guest, error)
	GetByQRCodeToken(ctx context.Context, qrCodeToken string) (*models.Guest, error)
	Update(ctx context.Context, guest *models.Guest) error
	Delete(ctx context.Context, id string) error
}

type RSVPRepository interface {
	Upsert(ctx context.Context, rsvp *models.RSVP) error
	ListByEventID(ctx context.Context, eventID string) ([]*models.RSVP, error)
	CountByEventID(ctx context.Context, eventID string) (int, error)
	ListByEventIDPaged(ctx context.Context, eventID string, limit, offset int) ([]*models.RSVP, error)
	GetByGuestID(ctx context.Context, guestID string) (*models.RSVP, error)
}

type GiftRepository interface {
	Create(ctx context.Context, gift *models.Gift) error
	ListByEventID(ctx context.Context, eventID string) ([]*models.Gift, error)
	CountByEventID(ctx context.Context, eventID string) (int, error)
	ListByEventIDPaged(ctx context.Context, eventID string, limit, offset int) ([]*models.Gift, error)
	GetByID(ctx context.Context, id string) (*models.Gift, error)
	Update(ctx context.Context, gift *models.Gift) error
	Delete(ctx context.Context, id string) error
}

type GalleryPhotoRepository interface {
	Create(ctx context.Context, photo *models.GalleryPhoto) error
	ListByEventID(ctx context.Context, eventID string) ([]*models.GalleryPhoto, error)
	CountByEventID(ctx context.Context, eventID string) (int, error)
	GetByID(ctx context.Context, id string) (*models.GalleryPhoto, error)
	Delete(ctx context.Context, id string) error
}

type GiftTransactionRepository interface {
	Create(ctx context.Context, transaction *models.GiftTransaction) error
	ListByEventID(ctx context.Context, eventID string) ([]*models.GiftTransaction, error)
	GetByID(ctx context.Context, id string) (*models.GiftTransaction, error)
	Update(ctx context.Context, transaction *models.GiftTransaction) error
	ExpirePendingBefore(ctx context.Context, cutoff, expiredAt time.Time) (int, error)
}
