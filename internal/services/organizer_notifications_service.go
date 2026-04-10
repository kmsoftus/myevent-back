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
	notifications    repositories.NotificationRepository
}

func NewOrganizerNotificationService(
	pushDeviceTokens repositories.PushDeviceTokenRepository,
	pushSender notifier.OrganizerPushSender,
	notifications repositories.NotificationRepository,
) *OrganizerNotificationService {
	if pushSender == nil {
		pushSender = notifier.NoopOrganizerPushSender{}
	}

	return &OrganizerNotificationService{
		pushDeviceTokens: pushDeviceTokens,
		pushSender:       pushSender,
		notifications:    notifications,
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

func (s *OrganizerNotificationService) SendPromotionalNotification(
	ctx context.Context,
	title, body string,
) (sent int, errs []error) {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)

	if title == "" {
		return 0, []error{NewValidationError("Informe o titulo da notificacao.", "notification_title_required",
			FieldError{Field: "title", Message: "Informe o titulo da notificacao."})}
	}
	if body == "" {
		return 0, []error{NewValidationError("Informe o corpo da notificacao.", "notification_body_required",
			FieldError{Field: "body", Message: "Informe o corpo da notificacao."})}
	}

	tokens, err := s.pushDeviceTokens.ListAll(ctx)
	if err != nil {
		return 0, []error{err}
	}

	msg := notifier.OrganizerPushMessage{
		Title: title,
		Body:  body,
		Data:  map[string]string{"type": "promotional"},
	}

	for _, deviceToken := range tokens {
		if err := s.pushSender.SendToDevice(ctx, deviceToken.Token, msg); err != nil {
			errs = append(errs, fmt.Errorf("token %s: %w", deviceToken.Token[:min(8, len(deviceToken.Token))], err))
			continue
		}
		sent++
	}

	return sent, errs
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// Save interno e listagem de tokens rodam em paralelo — sao independentes.
	type tokenResult struct {
		tokens []*models.PushDeviceToken
		err    error
	}
	tokenCh := make(chan tokenResult, 1)

	go func() {
		tokens, err := s.pushDeviceTokens.ListByUserID(ctx, userID)
		tokenCh <- tokenResult{tokens, err}
	}()

	if s.notifications != nil {
		_ = s.saveNotification(ctx, userID, message)
	}

	res := <-tokenCh
	if res.err != nil {
		return res.err
	}
	if len(res.tokens) == 0 {
		return nil
	}

	var firstErr error
	for _, deviceToken := range res.tokens {
		if err := s.pushSender.SendToDevice(ctx, deviceToken.Token, message); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (s *OrganizerNotificationService) saveNotification(
	ctx context.Context,
	userID string,
	message notifier.OrganizerPushMessage,
) error {
	notifType := message.Data["type"]
	if notifType == "" {
		notifType = "general"
	}

	n := &models.Notification{
		ID:        uuid.NewString(),
		UserID:    userID,
		Type:      notifType,
		Title:     message.Title,
		Body:      message.Body,
		Data:      message.Data,
		CreatedAt: time.Now().UTC(),
	}

	return s.notifications.Create(ctx, n)
}

// ListNotifications retorna as notificacoes internas do usuario.
func (s *OrganizerNotificationService) ListNotifications(
	ctx context.Context,
	userID string,
	limit, offset int,
) ([]*models.Notification, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	list, err := s.notifications.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	unread, err := s.notifications.CountUnreadByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return list, unread, nil
}

// MarkNotificationRead marca uma notificacao como lida.
func (s *OrganizerNotificationService) MarkNotificationRead(
	ctx context.Context,
	id, userID string,
) error {
	return s.notifications.MarkRead(ctx, id, userID)
}

// MarkAllNotificationsRead marca todas as notificacoes do usuario como lidas.
func (s *OrganizerNotificationService) MarkAllNotificationsRead(
	ctx context.Context,
	userID string,
) error {
	return s.notifications.MarkAllRead(ctx, userID)
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
