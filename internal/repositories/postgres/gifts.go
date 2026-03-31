package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type GiftRepository struct {
	db *pgxpool.Pool
}

func NewGiftRepository(db *pgxpool.Pool) *GiftRepository {
	return &GiftRepository{db: db}
}

const giftColumns = `id, event_id, title, description, image_url, value_cents,
	external_link, status, allow_reservation, allow_pix, created_at, updated_at`

func scanGift(row pgx.Row) (*models.Gift, error) {
	var g models.Gift
	err := row.Scan(
		&g.ID, &g.EventID, &g.Title, &g.Description, &g.ImageURL,
		&g.ValueCents, &g.ExternalLink, &g.Status,
		&g.AllowReservation, &g.AllowPix, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GiftRepository) Create(ctx context.Context, gift *models.Gift) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO gifts (id, event_id, title, description, image_url, value_cents,
			external_link, status, allow_reservation, allow_pix, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		gift.ID, gift.EventID, gift.Title, gift.Description, gift.ImageURL,
		gift.ValueCents, gift.ExternalLink, gift.Status,
		gift.AllowReservation, gift.AllowPix, gift.CreatedAt, gift.UpdatedAt,
	)
	return err
}

func (r *GiftRepository) ListByEventID(ctx context.Context, eventID string) ([]*models.Gift, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+giftColumns+` FROM gifts WHERE event_id = $1 ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []*models.Gift
	for rows.Next() {
		g, err := scanGift(rows)
		if err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}
	return gifts, rows.Err()
}

func (r *GiftRepository) CountByEventID(ctx context.Context, eventID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM gifts WHERE event_id = $1`, eventID).Scan(&count)
	return count, err
}

func (r *GiftRepository) ListByEventIDPaged(ctx context.Context, eventID string, limit, offset int) ([]*models.Gift, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+giftColumns+` FROM gifts WHERE event_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
		eventID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []*models.Gift
	for rows.Next() {
		g, err := scanGift(rows)
		if err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}
	return gifts, rows.Err()
}

func (r *GiftRepository) GetByID(ctx context.Context, id string) (*models.Gift, error) {
	row := r.db.QueryRow(ctx, `SELECT `+giftColumns+` FROM gifts WHERE id = $1`, id)
	g, err := scanGift(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *GiftRepository) Update(ctx context.Context, gift *models.Gift) error {
	_, err := r.db.Exec(ctx,
		`UPDATE gifts SET title=$1, description=$2, image_url=$3, value_cents=$4,
			external_link=$5, status=$6, allow_reservation=$7, allow_pix=$8, updated_at=$9
		 WHERE id=$10`,
		gift.Title, gift.Description, gift.ImageURL, gift.ValueCents,
		gift.ExternalLink, gift.Status, gift.AllowReservation, gift.AllowPix,
		gift.UpdatedAt, gift.ID,
	)
	return err
}

func (r *GiftRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM gifts WHERE id = $1`, id)
	return err
}
