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

type GiftTransactionDetails struct {
	Transaction *models.GiftTransaction
	Gift        *models.Gift
}

type GiftTransactionService struct {
	events       repositories.EventRepository
	gifts        repositories.GiftRepository
	transactions repositories.GiftTransactionRepository
	atomicTxs    atomicGiftTransactionRepository
}

func NewGiftTransactionService(
	events repositories.EventRepository,
	gifts repositories.GiftRepository,
	transactions repositories.GiftTransactionRepository,
) *GiftTransactionService {
	service := &GiftTransactionService{
		events:       events,
		gifts:        gifts,
		transactions: transactions,
	}

	if atomicTxs, ok := transactions.(atomicGiftTransactionRepository); ok {
		service.atomicTxs = atomicTxs
	}

	return service
}

func (s *GiftTransactionService) ReserveBySlug(ctx context.Context, slug, giftID string, input dto.CreateGiftTransactionRequest) (*GiftTransactionDetails, error) {
	return s.createPublicTransaction(ctx, slug, giftID, input, "reservation")
}

func (s *GiftTransactionService) RegisterPixBySlug(ctx context.Context, slug, giftID string, input dto.CreateGiftTransactionRequest) (*GiftTransactionDetails, error) {
	return s.createPublicTransaction(ctx, slug, giftID, input, "pix")
}

func (s *GiftTransactionService) ListByEvent(ctx context.Context, userID, eventID string) ([]GiftTransactionDetails, error) {
	if _, err := s.ensureEventOwnership(ctx, userID, eventID); err != nil {
		return nil, err
	}

	transactions, err := s.transactions.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	response := make([]GiftTransactionDetails, 0, len(transactions))
	for _, transaction := range transactions {
		gift, err := s.gifts.GetByID(ctx, transaction.GiftID)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				continue
			}
			return nil, err
		}

		response = append(response, GiftTransactionDetails{
			Transaction: transaction,
			Gift:        gift,
		})
	}

	return response, nil
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

	if err := validateGiftTransactionPayload(input.GuestName); err != nil {
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

	return &GiftTransactionDetails{Transaction: transaction, Gift: gift}, nil
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

func validateGiftTransactionPayload(guestName string) error {
	if strings.TrimSpace(guestName) == "" {
		return fmt.Errorf("%w: Informe o nome do convidado.", ErrValidation)
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
