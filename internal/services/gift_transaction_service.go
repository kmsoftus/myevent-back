package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type GiftTransactionDetails struct {
	Transaction *models.GiftTransaction
	Gift        *models.Gift
}

type GiftTransactionService struct {
	events                 repositories.EventRepository
	gifts                  repositories.GiftRepository
	transactions           repositories.GiftTransactionRepository
	atomicTxs              atomicGiftTransactionRepository
	pendingTTL             time.Duration
	organizerNotifications *OrganizerNotificationService
}

func NewGiftTransactionService(
	events repositories.EventRepository,
	gifts repositories.GiftRepository,
	transactions repositories.GiftTransactionRepository,
	pendingTTL time.Duration,
	organizerNotifications ...*OrganizerNotificationService,
) *GiftTransactionService {
	service := &GiftTransactionService{
		events:       events,
		gifts:        gifts,
		transactions: transactions,
		pendingTTL:   pendingTTL,
	}
	if len(organizerNotifications) > 0 {
		service.organizerNotifications = organizerNotifications[0]
	}

	if atomicTxs, ok := transactions.(atomicGiftTransactionRepository); ok {
		service.atomicTxs = atomicTxs
	}

	return service
}

func (s *GiftTransactionService) ExpirePending(ctx context.Context) (int, error) {
	if s.pendingTTL <= 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	return s.transactions.ExpirePendingBefore(ctx, now.Add(-s.pendingTTL), now)
}

func (s *GiftTransactionService) ReserveBySlug(ctx context.Context, slug, giftID string, input dto.CreateGiftTransactionRequest) (*GiftTransactionDetails, error) {
	return s.createPublicTransaction(ctx, slug, giftID, input, "reservation")
}

func (s *GiftTransactionService) RegisterPixBySlug(ctx context.Context, slug, giftID string, input dto.CreateGiftTransactionRequest) (*GiftTransactionDetails, error) {
	return s.createPublicTransaction(ctx, slug, giftID, input, "pix")
}

func (s *GiftTransactionService) ListByEvent(ctx context.Context, userID, eventID string, page, pageSize int) (*PagedResult[GiftTransactionDetails], error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	if page < 1 && pageSize < 1 {
		transactions, err := s.transactions.ListByEventID(ctx, eventID)
		if err != nil {
			return nil, err
		}

		items, err := s.buildTransactionDetails(ctx, transactions)
		if err != nil {
			return nil, err
		}

		total := len(transactions)
		totalPages := 0
		if total > 0 {
			totalPages = 1
		}

		return &PagedResult[GiftTransactionDetails]{
			Items:      items,
			Total:      total,
			Page:       1,
			PageSize:   total,
			TotalPages: totalPages,
		}, nil
	}

	pagination := normalizePagination(page, pageSize)

	total, err := s.transactions.CountByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	transactions, err := s.transactions.ListByEventIDPaged(ctx, eventID, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, err
	}

	response, err := s.buildTransactionDetails(ctx, transactions)
	if err != nil {
		return nil, err
	}

	return &PagedResult[GiftTransactionDetails]{
		Items:      response,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages(total, pagination.PageSize),
	}, nil
}

func (s *GiftTransactionService) Confirm(ctx context.Context, userID, eventID, transactionID string, input dto.UpdateGiftTransactionStatusRequest) (*GiftTransactionDetails, error) {
	if strings.TrimSpace(strings.ToLower(input.Status)) != "confirmed" {
		return nil, fmt.Errorf("%w: O status deve ser confirmado.", ErrValidation)
	}

	transaction, gift, err := s.getOwnedTransaction(ctx, userID, eventID, transactionID)
	if err != nil {
		return nil, err
	}
	if transaction.Status == "confirmed" {
		return nil, fmt.Errorf("%w: Esta transacao ja foi confirmada.", ErrConflict)
	}
	if transaction.Status == "canceled" {
		return nil, fmt.Errorf("%w: Nao e possivel confirmar uma transacao cancelada.", ErrConflict)
	}
	if transaction.Status == "expired" {
		return nil, fmt.Errorf("%w: Nao e possivel confirmar uma transacao expirada.", ErrConflict)
	}
	if transaction.Status != "pending" {
		return nil, fmt.Errorf("%w: Nao e possivel confirmar esta transacao no estado atual.", ErrConflict)
	}

	now := time.Now().UTC()
	transaction.Status = "confirmed"
	transaction.ConfirmedAt = &now
	transaction.UpdatedAt = now
	gift.Status = "confirmed"
	gift.UpdatedAt = now

	if s.atomicTxs != nil {
		if err := s.atomicTxs.UpdateTransactionAndGift(ctx, transaction, gift); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	} else {
		if err := s.transactions.Update(ctx, transaction); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		if err := s.gifts.Update(ctx, gift); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	return &GiftTransactionDetails{Transaction: transaction, Gift: gift}, nil
}

func (s *GiftTransactionService) Cancel(ctx context.Context, userID, eventID, transactionID string, input dto.UpdateGiftTransactionStatusRequest) (*GiftTransactionDetails, error) {
	if strings.TrimSpace(strings.ToLower(input.Status)) != "canceled" {
		return nil, fmt.Errorf("%w: O status deve ser cancelado.", ErrValidation)
	}

	transaction, gift, err := s.getOwnedTransaction(ctx, userID, eventID, transactionID)
	if err != nil {
		return nil, err
	}
	if transaction.Status == "canceled" {
		return nil, fmt.Errorf("%w: Esta transacao ja foi cancelada.", ErrConflict)
	}
	if transaction.Status == "confirmed" {
		return nil, fmt.Errorf("%w: Nao e possivel cancelar uma transacao ja confirmada.", ErrConflict)
	}
	if transaction.Status == "expired" {
		return nil, fmt.Errorf("%w: Esta transacao ja expirou.", ErrConflict)
	}
	if transaction.Status != "pending" {
		return nil, fmt.Errorf("%w: Nao e possivel cancelar esta transacao no estado atual.", ErrConflict)
	}

	now := time.Now().UTC()
	transaction.Status = "canceled"
	transaction.UpdatedAt = now
	gift.Status = "available"
	gift.UpdatedAt = now

	if s.atomicTxs != nil {
		if err := s.atomicTxs.UpdateTransactionAndGift(ctx, transaction, gift); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	} else {
		if err := s.transactions.Update(ctx, transaction); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		if err := s.gifts.Update(ctx, gift); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	return &GiftTransactionDetails{Transaction: transaction, Gift: gift}, nil
}

func (s *GiftTransactionService) createPublicTransaction(
	ctx context.Context,
	slug, giftID string,
	input dto.CreateGiftTransactionRequest,
	transactionType string,
) (*GiftTransactionDetails, error) {
	if _, err := s.ExpirePending(ctx); err != nil {
		return nil, err
	}

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

	gift, err := s.gifts.GetByID(ctx, giftID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if gift.EventID != event.ID {
		return nil, ErrNotFound
	}

	if err := validateGiftTransactionPayload(input); err != nil {
		return nil, err
	}

	if gift.Status != "available" {
		return nil, fmt.Errorf("%w: Este presente nao esta disponivel.", ErrConflict)
	}

	nextGiftStatus, err := validateGiftAction(gift, transactionType)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	transaction := &models.GiftTransaction{
		ID:           uuid.NewString(),
		GiftID:       gift.ID,
		EventID:      event.ID,
		GuestName:    strings.TrimSpace(input.GuestName),
		GuestContact: strings.TrimSpace(input.GuestContact),
		Type:         transactionType,
		Status:       "pending",
		Message:      strings.TrimSpace(input.Message),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	gift.Status = nextGiftStatus
	gift.UpdatedAt = now

	if s.atomicTxs != nil {
		gift, err = s.atomicTxs.CreatePendingForGift(ctx, transaction, nextGiftStatus)
		if err != nil {
			switch {
			case errors.Is(err, repositories.ErrNotFound):
				return nil, ErrNotFound
			case errors.Is(err, repositories.ErrConflict):
				return nil, fmt.Errorf("%w: Este presente nao esta disponivel.", ErrConflict)
			default:
				return nil, err
			}
		}
	} else {
		if err := s.transactions.Create(ctx, transaction); err != nil {
			if errors.Is(err, repositories.ErrConflict) {
				return nil, ErrConflict
			}
			return nil, err
		}
		if err := s.gifts.Update(ctx, gift); err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	if s.organizerNotifications != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), asyncNotificationTimeout)
			defer cancel()

			if err := s.organizerNotifications.NotifyGiftReserved(notifyCtx, event, gift, transaction); err != nil {
				log.Printf(
					"organizer gift push notification failed for event %s gift %s transaction %s: %v",
					event.ID,
					gift.ID,
					transaction.ID,
					err,
				)
			}
		}()
	}

	return &GiftTransactionDetails{Transaction: transaction, Gift: gift}, nil
}

func (s *GiftTransactionService) buildTransactionDetails(
	ctx context.Context,
	transactions []*models.GiftTransaction,
) ([]GiftTransactionDetails, error) {
	if len(transactions) == 0 {
		return []GiftTransactionDetails{}, nil
	}

	// Batch-load gifts in a single query instead of one query per transaction.
	giftIDs := make([]string, len(transactions))
	for i, t := range transactions {
		giftIDs[i] = t.GiftID
	}
	giftSlice, err := s.gifts.GetByIDs(ctx, giftIDs)
	if err != nil {
		return nil, err
	}
	giftByID := make(map[string]*models.Gift, len(giftSlice))
	for _, g := range giftSlice {
		giftByID[g.ID] = g
	}

	response := make([]GiftTransactionDetails, 0, len(transactions))
	for _, transaction := range transactions {
		gift, ok := giftByID[transaction.GiftID]
		if !ok {
			continue
		}
		response = append(response, GiftTransactionDetails{
			Transaction: transaction,
			Gift:        gift,
		})
	}

	return response, nil
}

func (s *GiftTransactionService) getOwnedTransaction(ctx context.Context, userID, eventID, transactionID string) (*models.GiftTransaction, *models.Gift, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, nil, err
	}

	transaction, err := s.transactions.GetByID(ctx, transactionID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	if transaction.EventID != eventID {
		return nil, nil, ErrNotFound
	}

	gift, err := s.gifts.GetByID(ctx, transaction.GiftID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	if gift.EventID != eventID {
		return nil, nil, ErrNotFound
	}

	return transaction, gift, nil
}

func (s *GiftTransactionService) ensureEventOwnership(ctx context.Context, userID, eventID string) (*models.Event, error) {
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

func validateGiftTransactionPayload(input dto.CreateGiftTransactionRequest) error {
	if strings.TrimSpace(input.GuestName) == "" {
		return fmt.Errorf("%w: Informe o nome do convidado.", ErrValidation)
	}
	if err := validateTextMaxLength("guest_contact", "contato do convidado", input.GuestContact, maxGiftTransactionGuestContactLen, "gift_transaction_guest_contact_too_long"); err != nil {
		return err
	}
	if err := validateTextMaxLength("message", "mensagem", input.Message, maxGiftTransactionMessageLength, "gift_transaction_message_too_long"); err != nil {
		return err
	}
	return nil
}

func validateGiftAction(gift *models.Gift, transactionType string) (string, error) {
	switch transactionType {
	case "reservation":
		if !gift.AllowReservation {
			return "", fmt.Errorf("%w: Este presente nao aceita reservas.", ErrValidation)
		}
		return "reserved", nil
	case "pix":
		if !gift.AllowPix {
			return "", fmt.Errorf("%w: Este presente nao aceita pagamento via Pix.", ErrValidation)
		}
		return "pending_payment", nil
	default:
		return "", fmt.Errorf("%w: Tipo de transacao invalido.", ErrValidation)
	}
}
