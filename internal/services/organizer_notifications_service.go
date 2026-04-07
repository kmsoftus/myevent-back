package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/models"
	"myevent-back/internal/notifier"
	"myevent-back/internal/repositories"
)

type OrganizerNotificationService struct {
	pushDeviceTokens repositories.PushDeviceTokenRepository
	pushSender       notifier.OrganizerPushSender
}

func NewOrganizerNotificationService(
	pushDeviceTokens repositories.PushDeviceTokenRepository,
	pushSender notifier.OrganizerPushSender,
) *OrganizerNotificationService {
	if pushSender == nil {
		pushSender = notifier.NoopOrganizerPushSender{}
	}

	return &OrganizerNotificationService{
		pushDeviceTokens: pushDeviceTokens,
		pushSender:       pushSender,
	}
}

func (s *OrganizerNotificationService) RegisterDeviceToken(
	ctx context.Context,
	userID, token, platform string,
) error {
	userID = strings.TrimSpace(userID)
	token = strings.TrimSpace(token)
	platform = normalizeDevicePlatform(platform)

	if userID == "" {
		return NewUnauthorizedError(
			"Sessao invalida. Faca login novamente.",
			"auth_session_invalid",
		)
	}
	if token == "" {
		return NewValidationError(
			"Informe o token do dispositivo.",
			"push_device_token_required",
			FieldError{Field: "token", Message: "Informe o token do dispositivo."},
		)
	}
	if len(token) < 20 {
		return NewValidationError(
			"Token do dispositivo invalido.",
			"push_device_token_invalid",
			FieldError{Field: "token", Message: "Token do dispositivo invalido."},
		)
	}

	now := time.Now().UTC()
	deviceToken := &models.PushDeviceToken{
		ID:         uuid.NewString(),
		UserID:     userID,
		Token:      token,
		Platform:   platform,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastSeenAt: now,
	}

	if err := s.pushDeviceTokens.Upsert(ctx, deviceToken); err != nil {
		if err == repositories.ErrNotFound {
			return NewUnauthorizedError(
				"Sessao invalida. Faca login novamente.",
				"auth_session_invalid",
			)
		}
		return err
	}

	return nil
}

func (s *OrganizerNotificationService) NotifyNewRSVP(
	ctx context.Context,
	event *models.Event,
	guest *models.Guest,
	rsvp *models.RSVP,
) error {
	if event == nil || guest == nil || rsvp == nil {
		return nil
	}

	title := "Novo RSVP"
	body := fmt.Sprintf(
		"%s respondeu %s no evento %s.",
		guest.Name,
		humanizeRSVPStatus(rsvp.Status),
		event.Title,
	)

	return s.notifyUser(ctx, event.UserID, notifier.OrganizerPushMessage{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":     "rsvp_new",
			"event_id": event.ID,
			"guest_id": guest.ID,
			"status":   rsvp.Status,
		},
	})
}

func (s *OrganizerNotificationService) NotifyGiftReserved(
	ctx context.Context,
	event *models.Event,
	gift *models.Gift,
	transaction *models.GiftTransaction,
) error {
	if event == nil || gift == nil || transaction == nil {
		return nil
	}

	title := "Presente reservado"
	body := fmt.Sprintf(
		"%s escolheu o presente %s no evento %s.",
		transaction.GuestName,
		gift.Title,
		event.Title,
	)
	if transaction.Type == "pix" {
		title = "Presente com Pix"
		body = fmt.Sprintf(
			"%s informou pagamento Pix para %s no evento %s.",
			transaction.GuestName,
			gift.Title,
			event.Title,
		)
	}

	return s.notifyUser(ctx, event.UserID, notifier.OrganizerPushMessage{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":           "gift_reserved",
			"event_id":       event.ID,
			"gift_id":        gift.ID,
			"transaction_id": transaction.ID,
			"transaction":    transaction.Type,
		},
	})
}

func (s *OrganizerNotificationService) notifyUser(
	ctx context.Context,
	userID string,
	message notifier.OrganizerPushMessage,
) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}

	tokens, err := s.pushDeviceTokens.ListByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	var firstErr error
	for _, deviceToken := range tokens {
		if err := s.pushSender.SendToDevice(ctx, deviceToken.Token, message); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func normalizeDevicePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "android":
		return "android"
	case "ios":
		return "ios"
	case "web":
		return "web"
	default:
		return "unknown"
	}
}

func humanizeRSVPStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "confirmed":
		return "confirmou presenca"
	case "declined":
		return "recusou o convite"
	default:
		return "atualizou o RSVP"
	}
}
