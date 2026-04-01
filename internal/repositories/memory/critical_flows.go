package memory

import (
	"context"
	"strings"
	"time"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

func (r *guestRepository) FindOrCreateOpenRSVPGuest(_ context.Context, guest *models.Guest) (*models.Guest, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.events[guest.EventID]; !ok {
		return nil, repositories.ErrNotFound
	}

	normalizedName := strings.ToLower(strings.TrimSpace(guest.Name))
	for guestID := range r.store.guestIDsByEvent[guest.EventID] {
		existing := r.store.guests[guestID]
		if strings.ToLower(strings.TrimSpace(existing.Name)) == normalizedName {
			return cloneGuest(existing), nil
		}
	}

	if _, exists := r.store.guests[guest.ID]; exists {
		return nil, repositories.ErrConflict
	}
	if _, exists := r.store.guestByInvite[strings.ToUpper(strings.TrimSpace(guest.InviteCode))]; exists {
		return nil, repositories.ErrConflict
	}
	if _, exists := r.store.guestByQRToken[guest.QRCodeToken]; exists {
		return nil, repositories.ErrConflict
	}
	for guestID := range r.store.guestIDsByEvent[guest.EventID] {
		existing := r.store.guests[guestID]
		if existing.ShortCode == guest.ShortCode && guest.ShortCode != "" {
			return nil, repositories.ErrConflict
		}
	}

	r.store.guests[guest.ID] = cloneGuest(guest)
	r.store.guestByInvite[strings.ToUpper(strings.TrimSpace(guest.InviteCode))] = guest.ID
	r.store.guestByQRToken[guest.QRCodeToken] = guest.ID

	if _, ok := r.store.guestIDsByEvent[guest.EventID]; !ok {
		r.store.guestIDsByEvent[guest.EventID] = make(map[string]struct{})
	}
	r.store.guestIDsByEvent[guest.EventID][guest.ID] = struct{}{}

	return cloneGuest(guest), nil
}

func (r *giftTransactionRepository) CreatePendingForGift(_ context.Context, transaction *models.GiftTransaction, nextGiftStatus string) (*models.Gift, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	gift, ok := r.store.gifts[transaction.GiftID]
	if !ok || gift.EventID != transaction.EventID {
		return nil, repositories.ErrNotFound
	}
	if gift.Status != "available" {
		return nil, repositories.ErrConflict
	}
	if _, exists := r.store.giftTransactions[transaction.ID]; exists {
		return nil, repositories.ErrConflict
	}

	r.store.giftTransactions[transaction.ID] = cloneGiftTransaction(transaction)
	if _, ok := r.store.giftTransactionIDsByEvent[transaction.EventID]; !ok {
		r.store.giftTransactionIDsByEvent[transaction.EventID] = make(map[string]struct{})
	}
	r.store.giftTransactionIDsByEvent[transaction.EventID][transaction.ID] = struct{}{}

	gift.Status = nextGiftStatus
	gift.UpdatedAt = transaction.UpdatedAt

	return cloneGift(gift), nil
}

func (r *giftTransactionRepository) UpdateTransactionAndGift(_ context.Context, transaction *models.GiftTransaction, gift *models.Gift) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	currentTransaction, ok := r.store.giftTransactions[transaction.ID]
	if !ok {
		return repositories.ErrNotFound
	}
	currentGift, ok := r.store.gifts[gift.ID]
	if !ok {
		return repositories.ErrNotFound
	}

	currentTransaction.Status = transaction.Status
	currentTransaction.ConfirmedAt = transaction.ConfirmedAt
	currentTransaction.UpdatedAt = transaction.UpdatedAt

	currentGift.Status = gift.Status
	currentGift.UpdatedAt = gift.UpdatedAt

	return nil
}

func (r *giftTransactionRepository) ExpirePendingBefore(_ context.Context, cutoff, expiredAt time.Time) (int, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	expiredCount := 0
	for _, transaction := range r.store.giftTransactions {
		if transaction.Status != "pending" || transaction.CreatedAt.After(cutoff) {
			continue
		}

		gift, ok := r.store.gifts[transaction.GiftID]
		if !ok {
			continue
		}
		if gift.Status != "reserved" && gift.Status != "pending_payment" {
			continue
		}

		transaction.Status = "expired"
		transaction.UpdatedAt = expiredAt
		gift.Status = "available"
		gift.UpdatedAt = expiredAt
		expiredCount++
	}

	return expiredCount, nil
}
