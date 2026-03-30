package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type GiftTransactionRepository struct {
	db *pgxpool.Pool
}

func NewGiftTransactionRepository(db *pgxpool.Pool) *GiftTransactionRepository {
	return &GiftTransactionRepository{db: db}
}

const txColumns = `id, gift_id, event_id, guest_name, guest_contact, type, status, message,
	confirmed_at, created_at, updated_at`

func scanTransaction(row pgx.Row) (*models.GiftTransaction, error) {
	var t models.GiftTransaction
	err := row.Scan(
		&t.ID, &t.GiftID, &t.EventID, &t.GuestName, &t.GuestContact,
		&t.Type, &t.Status, &t.Message, &t.ConfirmedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *GiftTransactionRepository) Create(ctx context.Context, t *models.GiftTransaction) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO gift_transactions (id, gift_id, event_id, guest_name, guest_contact, type, status, message,
			confirmed_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		t.ID, t.GiftID, t.EventID, t.GuestName, t.GuestContact,
		t.Type, t.Status, t.Message, t.ConfirmedAt, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *GiftTransactionRepository) ListByEventID(ctx context.Context, eventID string) ([]*models.GiftTransaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+txColumns+` FROM gift_transactions WHERE event_id = $1 ORDER BY created_at DESC`, eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*models.GiftTransaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

func (r *GiftTransactionRepository) GetByID(ctx context.Context, id string) (*models.GiftTransaction, error) {
	row := r.db.QueryRow(ctx, `SELECT `+txColumns+` FROM gift_transactions WHERE id = $1`, id)
	t, err := scanTransaction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *GiftTransactionRepository) Update(ctx context.Context, t *models.GiftTransaction) error {
	_, err := r.db.Exec(ctx,
		`UPDATE gift_transactions SET status=$1, confirmed_at=$2, updated_at=$3 WHERE id=$4`,
		t.Status, t.ConfirmedAt, t.UpdatedAt, t.ID,
	)
	return err
}
