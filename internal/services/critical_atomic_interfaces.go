package services

import (
	"context"
	"time"

	"myevent-back/internal/models"
)

type atomicGiftTransactionRepository interface {
	CreatePendingForGift(ctx context.Context, transaction *models.GiftTransaction, nextGiftStatus string) (*models.Gift, error)
	UpdateTransactionAndGift(ctx context.Context, transaction *models.GiftTransaction, gift *models.Gift) error
	ExpirePendingBefore(ctx context.Context, cutoff, expiredAt time.Time) (int, error)
}

type atomicOpenRSVPGuestRepository interface {
	FindOrCreateOpenRSVPGuest(ctx context.Context, guest *models.Guest) (*models.Guest, error)
}
