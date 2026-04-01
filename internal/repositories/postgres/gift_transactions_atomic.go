package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

func (r *GiftTransactionRepository) CreatePendingForGift(ctx context.Context, transaction *models.GiftTransaction, nextGiftStatus string) (*models.Gift, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	gift, err := scanGift(tx.QueryRow(ctx, `SELECT `+giftColumns+` FROM gifts WHERE id = $1 FOR UPDATE`, transaction.GiftID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	if gift.EventID != transaction.EventID {
		return nil, repositories.ErrNotFound
	}
	if gift.Status != "available" {
		return nil, repositories.ErrConflict
	}

	gift.Status = nextGiftStatus
	gift.UpdatedAt = transaction.UpdatedAt

	if _, err := tx.Exec(ctx,
		`INSERT INTO gift_transactions (id, gift_id, event_id, guest_name, guest_contact, type, status, message,
			confirmed_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		transaction.ID, transaction.GiftID, transaction.EventID, transaction.GuestName, transaction.GuestContact,
		transaction.Type, transaction.Status, transaction.Message, transaction.ConfirmedAt, transaction.CreatedAt, transaction.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, repositories.ErrConflict
		}
		return nil, err
	}

	tag, err := tx.Exec(ctx, `UPDATE gifts SET status = $1, updated_at = $2 WHERE id = $3`, gift.Status, gift.UpdatedAt, gift.ID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, repositories.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return gift, nil
}

func (r *GiftTransactionRepository) UpdateTransactionAndGift(ctx context.Context, transaction *models.GiftTransaction, gift *models.Gift) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := scanTransaction(tx.QueryRow(ctx, `SELECT `+txColumns+` FROM gift_transactions WHERE id = $1 FOR UPDATE`, transaction.ID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repositories.ErrNotFound
		}
		return err
	}
	if _, err := scanGift(tx.QueryRow(ctx, `SELECT `+giftColumns+` FROM gifts WHERE id = $1 FOR UPDATE`, gift.ID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repositories.ErrNotFound
		}
		return err
	}

	transactionTag, err := tx.Exec(ctx,
		`UPDATE gift_transactions SET status = $1, confirmed_at = $2, updated_at = $3 WHERE id = $4`,
		transaction.Status, transaction.ConfirmedAt, transaction.UpdatedAt, transaction.ID,
	)
	if err != nil {
		return err
	}
	if transactionTag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	giftTag, err := tx.Exec(ctx,
		`UPDATE gifts SET status = $1, updated_at = $2 WHERE id = $3`,
		gift.Status, gift.UpdatedAt, gift.ID,
	)
	if err != nil {
		return err
	}
	if giftTag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	return tx.Commit(ctx)
}
