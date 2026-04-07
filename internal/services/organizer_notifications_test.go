package services

import (
	"context"
	"testing"
	"time"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/notifier"
	"myevent-back/internal/repositories/memory"
)

func TestOrganizerNotificationServiceRegisterDeviceToken(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Now().UTC()

	user := &models.User{
		ID:           "user-notify-1",
		Name:         "Kaleb",
		Email:        "kaleb-notify@example.com",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Users().Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	service := NewOrganizerNotificationService(
		store.PushDeviceTokens(),
		notifier.NoopOrganizerPushSender{},
	)

	if err := service.RegisterDeviceToken(ctx, user.ID, "abcdefghijklmnopqrstuvwxyz123456", "IOS"); err != nil {
		t.Fatalf("register device token: %v", err)
	}

	tokens, err := store.PushDeviceTokens().ListByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].Platform != "ios" {
		t.Fatalf("expected normalized platform ios, got %q", tokens[0].Platform)
	}
}

func TestRSVPServiceSubmitBySlugSendsOrganizerPush(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Now().UTC()

	owner := &models.User{
		ID:           "owner-rsvp",
		Name:         "Owner",
		Email:        "owner-rsvp@example.com",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Users().Create(ctx, owner); err != nil {
		t.Fatalf("create owner user: %v", err)
	}

	pushSender := &captureOrganizerPushSender{}
	notificationService := NewOrganizerNotificationService(store.PushDeviceTokens(), pushSender)
	if err := notificationService.RegisterDeviceToken(ctx, owner.ID, "token-owner-rsvp-abcdefghijklmnopqrstuvwxyz", "android"); err != nil {
		t.Fatalf("register owner token: %v", err)
	}

	event := &models.Event{
		ID:        "event-rsvp",
		UserID:    owner.ID,
		Title:     "Casamento",
		Slug:      "casamento-rsvp",
		Status:    "published",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	guest := &models.Guest{
		ID:          "guest-rsvp",
		EventID:     event.ID,
		Name:        "Maria",
		InviteCode:  "ABC12345",
		ShortCode:   "123456",
		QRCodeToken: "qr-rsvp-123",
		RSVPStatus:  "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Guests().Create(ctx, guest); err != nil {
		t.Fatalf("create guest: %v", err)
	}

	service := NewRSVPService(store.Events(), store.Guests(), store.RSVPs(), 0, notificationService)
	if _, err := service.SubmitBySlug(ctx, event.Slug, dto.CreateRSVPRequest{
		GuestIdentifier: guest.InviteCode,
		Status:          "confirmed",
	}); err != nil {
		t.Fatalf("submit RSVP: %v", err)
	}

	if len(pushSender.sent) != 1 {
		t.Fatalf("expected 1 push notification, got %d", len(pushSender.sent))
	}
	if pushSender.sent[0].message.Data["type"] != "rsvp_new" {
		t.Fatalf("expected push type rsvp_new, got %q", pushSender.sent[0].message.Data["type"])
	}
}

func TestGiftTransactionServiceReserveBySlugSendsOrganizerPush(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Now().UTC()

	owner := &models.User{
		ID:           "owner-gift",
		Name:         "Owner",
		Email:        "owner-gift@example.com",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Users().Create(ctx, owner); err != nil {
		t.Fatalf("create owner user: %v", err)
	}

	pushSender := &captureOrganizerPushSender{}
	notificationService := NewOrganizerNotificationService(store.PushDeviceTokens(), pushSender)
	if err := notificationService.RegisterDeviceToken(ctx, owner.ID, "token-owner-gift-abcdefghijklmnopqrstuvwxyz", "android"); err != nil {
		t.Fatalf("register owner token: %v", err)
	}

	event := &models.Event{
		ID:        "event-gift",
		UserID:    owner.ID,
		Title:     "Cha de Casa Nova",
		Slug:      "cha-casa-nova",
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

	service := NewGiftTransactionService(
		store.Events(),
		store.Gifts(),
		store.GiftTransactions(),
		time.Hour,
		notificationService,
	)
	if _, err := service.ReserveBySlug(ctx, event.Slug, gift.ID, dto.CreateGiftTransactionRequest{
		GuestName: "Carlos",
	}); err != nil {
		t.Fatalf("reserve gift: %v", err)
	}

	if len(pushSender.sent) != 1 {
		t.Fatalf("expected 1 push notification, got %d", len(pushSender.sent))
	}
	if pushSender.sent[0].message.Data["type"] != "gift_reserved" {
		t.Fatalf("expected push type gift_reserved, got %q", pushSender.sent[0].message.Data["type"])
	}
}

type captureOrganizerPushSender struct {
	sent []captureOrganizerPushMessage
}

type captureOrganizerPushMessage struct {
	token   string
	message notifier.OrganizerPushMessage
}

func (s *captureOrganizerPushSender) SendToDevice(_ context.Context, token string, message notifier.OrganizerPushMessage) error {
	s.sent = append(s.sent, captureOrganizerPushMessage{
		token:   token,
		message: message,
	})
	return nil
}
