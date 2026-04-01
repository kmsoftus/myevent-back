package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"myevent-back/internal/auth"
	"myevent-back/internal/dto"
	"myevent-back/internal/mailer"
	"myevent-back/internal/models"
	"myevent-back/internal/notifier"
	"myevent-back/internal/repositories/memory"
)

func TestAuthServiceForgotPasswordSucceedsWhenEmailSenderFails(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	passwordHash, err := auth.HashPassword("Senha123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:           "user-1",
		Name:         "Kaleb",
		Email:        "kaleb@example.com",
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Users().Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	service := NewAuthService(
		store.Users(),
		store.PasswordResetTokens(),
		auth.NewJWTManager("test-secret", time.Hour),
		time.Hour,
		"http://localhost:3000/redefinir-senha",
		failingPasswordResetSender{},
		notifier.NoopRegistrationSender{},
	)

	message, err := service.ForgotPassword(ctx, user.Email)
	if err != nil {
		t.Fatalf("forgot password returned error: %v", err)
	}
	if message != forgotPasswordSuccessMessage {
		t.Fatalf("expected success message %q, got %q", forgotPasswordSuccessMessage, message)
	}
}

func TestGiftTransactionServiceExpirePendingReleasesGift(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions(), time.Hour)

	now := time.Now().UTC()
	event := &models.Event{
		ID:        "event-1",
		UserID:    "user-1",
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
		ID:               "gift-1",
		EventID:          event.ID,
		Title:            "Panela",
		Status:           "reserved",
		AllowReservation: true,
		AllowPix:         true,
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	if err := store.Gifts().Create(ctx, gift); err != nil {
		t.Fatalf("create gift: %v", err)
	}

	transaction := &models.GiftTransaction{
		ID:        "tx-1",
		GiftID:    gift.ID,
		EventID:   event.ID,
		GuestName: "Maria",
		Type:      "reservation",
		Status:    "pending",
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	}
	if err := store.GiftTransactions().Create(ctx, transaction); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	expired, err := service.ExpirePending(ctx)
	if err != nil {
		t.Fatalf("expire pending: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expected 1 expired transaction, got %d", expired)
	}

	updatedTransaction, err := store.GiftTransactions().GetByID(ctx, transaction.ID)
	if err != nil {
		t.Fatalf("get transaction: %v", err)
	}
	if updatedTransaction.Status != "expired" {
		t.Fatalf("expected transaction status expired, got %s", updatedTransaction.Status)
	}

	updatedGift, err := store.Gifts().GetByID(ctx, gift.ID)
	if err != nil {
		t.Fatalf("get gift: %v", err)
	}
	if updatedGift.Status != "available" {
		t.Fatalf("expected gift status available, got %s", updatedGift.Status)
	}
}

func TestRSVPServiceOpenRSVPUsesConservativeDefaultMaxCompanions(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewRSVPService(store.Events(), store.Guests(), store.RSVPs(), 0)

	now := time.Now().UTC()
	event := &models.Event{
		ID:        "event-open",
		UserID:    "user-1",
		Title:     "Cha",
		Slug:      "cha",
		Status:    "published",
		OpenRSVP:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	_, err := service.SubmitBySlug(ctx, event.Slug, dto.CreateRSVPRequest{
		GuestIdentifier: "Patricia",
		Status:          "confirmed",
		CompanionsCount: 1,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for unexpected companions, got %v", err)
	}

	details, err := service.SubmitBySlug(ctx, event.Slug, dto.CreateRSVPRequest{
		GuestIdentifier: "Patricia",
		Status:          "confirmed",
		CompanionsCount: 0,
	})
	if err != nil {
		t.Fatalf("submit RSVP without companions: %v", err)
	}
	if details.Guest.MaxCompanions != 0 {
		t.Fatalf("expected default max companions 0, got %d", details.Guest.MaxCompanions)
	}
}

func TestP2TextLengthValidations(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Now().UTC()

	event := &models.Event{
		ID:        "event-validation",
		UserID:    "user-1",
		Title:     "Evento",
		Slug:      "evento",
		Status:    "published",
		OpenRSVP:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	gift := &models.Gift{
		ID:               "gift-validation",
		EventID:          event.ID,
		Title:            "Gift",
		Status:           "available",
		AllowReservation: true,
		AllowPix:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Gifts().Create(ctx, gift); err != nil {
		t.Fatalf("create gift: %v", err)
	}

	eventService := NewEventService(store.Events())
	guestService := NewGuestService(store.Events(), store.Guests())
	rsvpService := NewRSVPService(store.Events(), store.Guests(), store.RSVPs(), 0)
	giftService := NewGiftService(store.Events(), store.Gifts())
	giftTransactionService := NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions(), time.Hour)

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "event description",
			run: func() error {
				_, err := eventService.Create(ctx, event.UserID, dto.CreateEventRequest{
					Title:       "Descricao longa",
					Slug:        "descricao-longa",
					Description: repeated("a", maxEventDescriptionLength+1),
				})
				return err
			},
		},
		{
			name: "gift description",
			run: func() error {
				_, err := giftService.Create(ctx, event.UserID, event.ID, dto.CreateGiftRequest{
					Title:       "Presente",
					Description: repeated("a", maxGiftDescriptionLength+1),
				})
				return err
			},
		},
		{
			name: "guest notes",
			run: func() error {
				_, err := guestService.Create(ctx, event.UserID, event.ID, dto.CreateGuestRequest{
					Name:  "Convidado",
					Notes: repeated("a", maxGuestNotesLength+1),
				})
				return err
			},
		},
		{
			name: "rsvp message",
			run: func() error {
				_, err := rsvpService.SubmitBySlug(ctx, event.Slug, dto.CreateRSVPRequest{
					GuestIdentifier: "Open RSVP",
					Status:          "confirmed",
					Message:         repeated("a", maxRSVPMessageLength+1),
				})
				return err
			},
		},
		{
			name: "gift transaction message",
			run: func() error {
				_, err := giftTransactionService.ReserveBySlug(ctx, event.Slug, gift.ID, dto.CreateGiftTransactionRequest{
					GuestName: "Convidado",
					Message:   repeated("a", maxGiftTransactionMessageLength+1),
				})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.run()
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

type failingPasswordResetSender struct{}

func (failingPasswordResetSender) SendPasswordReset(context.Context, mailer.PasswordResetMessage) error {
	return errors.New("smtp down")
}

func repeated(char string, count int) string {
	result := ""
	for range count {
		result += char
	}
	return result
}
