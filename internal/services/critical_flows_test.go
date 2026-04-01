package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories/memory"
)

func TestGiftTransactionServiceReserveBySlugPreventsDoubleBooking(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions())

	now := time.Now().UTC()
	event := &models.Event{
		ID:        "event-1",
		UserID:    "user-1",
		Title:     "Cha Bar",
		Slug:      "cha-bar",
		Status:    "published",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	gift := &models.Gift{
		ID:               "gift-1",
		EventID:          event.ID,
		Title:            "Jogo de panelas",
		Status:           "available",
		AllowReservation: true,
		AllowPix:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Gifts().Create(ctx, gift); err != nil {
		t.Fatalf("create gift: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)

	for i := range 2 {
		go func(index int) {
			<-start
			_, err := service.ReserveBySlug(ctx, event.Slug, gift.ID, dto.CreateGiftTransactionRequest{
				GuestName: fmt.Sprintf("Convidado %d", index),
			})
			errs <- err
		}(i)
	}

	close(start)

	var successCount int
	var conflictCount int
	for range 2 {
		err := <-errs
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, ErrConflict):
			conflictCount++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("expected 1 success and 1 conflict, got success=%d conflict=%d", successCount, conflictCount)
	}

	transactions, err := store.GiftTransactions().ListByEventID(ctx, event.ID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}

	updatedGift, err := store.Gifts().GetByID(ctx, gift.ID)
	if err != nil {
		t.Fatalf("get gift: %v", err)
	}
	if updatedGift.Status != "reserved" {
		t.Fatalf("expected gift status reserved, got %s", updatedGift.Status)
	}
}

func TestRSVPServiceSubmitBySlugOpenRSVPAvoidsDuplicateGuest(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewRSVPService(store.Events(), store.Guests(), store.RSVPs())

	now := time.Now().UTC()
	event := &models.Event{
		ID:        "event-open-rsvp",
		UserID:    "user-1",
		Title:     "Aniversario",
		Slug:      "aniversario",
		Status:    "published",
		OpenRSVP:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	start := make(chan struct{})
	type result struct {
		details *RSVPDetails
		err     error
	}
	results := make(chan result, 2)

	for range 2 {
		go func() {
			<-start
			details, err := service.SubmitBySlug(ctx, event.Slug, dto.CreateRSVPRequest{
				GuestIdentifier: "Patricia",
				Status:          "confirmed",
			})
			results <- result{details: details, err: err}
		}()
	}

	close(start)

	guestIDs := make(map[string]struct{})
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Fatalf("submit RSVP: %v", result.err)
		}
		guestIDs[result.details.Guest.ID] = struct{}{}
	}

	if len(guestIDs) != 1 {
		t.Fatalf("expected both submissions to resolve to the same guest, got %d guest IDs", len(guestIDs))
	}

	guests, err := store.Guests().ListByEventID(ctx, event.ID)
	if err != nil {
		t.Fatalf("list guests: %v", err)
	}
	if len(guests) != 1 {
		t.Fatalf("expected 1 guest, got %d", len(guests))
	}

	rsvps, err := store.RSVPs().ListByEventID(ctx, event.ID)
	if err != nil {
		t.Fatalf("list rsvps: %v", err)
	}
	if len(rsvps) != 1 {
		t.Fatalf("expected 1 RSVP, got %d", len(rsvps))
	}
}

func TestGiftTransactionServiceConfirmUpdatesGiftAndTransactionAtomically(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions())

	now := time.Now().UTC()
	event := &models.Event{
		ID:        "event-confirm",
		UserID:    "owner-1",
		Title:     "Casamento",
		Slug:      "casamento",
		Status:    "published",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	gift := &models.Gift{
		ID:               "gift-confirm",
		EventID:          event.ID,
		Title:            "Air fryer",
		Status:           "reserved",
		AllowReservation: true,
		AllowPix:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Gifts().Create(ctx, gift); err != nil {
		t.Fatalf("create gift: %v", err)
	}

	transaction := &models.GiftTransaction{
		ID:        "tx-confirm",
		GiftID:    gift.ID,
		EventID:   event.ID,
		GuestName: "Marina",
		Type:      "reservation",
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.GiftTransactions().Create(ctx, transaction); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	details, err := service.Confirm(ctx, event.UserID, event.ID, transaction.ID, dto.UpdateGiftTransactionStatusRequest{
		Status: "confirmed",
	})
	if err != nil {
		t.Fatalf("confirm transaction: %v", err)
	}
	if details.Transaction.Status != "confirmed" {
		t.Fatalf("expected transaction status confirmed, got %s", details.Transaction.Status)
	}
	if details.Gift.Status != "confirmed" {
		t.Fatalf("expected gift status confirmed, got %s", details.Gift.Status)
	}
	if details.Transaction.ConfirmedAt == nil {
		t.Fatal("expected transaction confirmation timestamp to be set")
	}
}
