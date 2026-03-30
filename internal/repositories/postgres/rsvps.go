package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
)

type RSVPRepository struct {
	db *pgxpool.Pool
}

func NewRSVPRepository(db *pgxpool.Pool) *RSVPRepository {
	return &RSVPRepository{db: db}
}

func (r *RSVPRepository) Upsert(ctx context.Context, rsvp *models.RSVP) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO rsvps (id, event_id, guest_id, status, companions_count, message, responded_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (guest_id) DO UPDATE SET
			status = EXCLUDED.status,
			companions_count = EXCLUDED.companions_count,
			message = EXCLUDED.message,
			responded_at = EXCLUDED.responded_at,
			updated_at = EXCLUDED.updated_at`,
		rsvp.ID, rsvp.EventID, rsvp.GuestID, rsvp.Status,
		rsvp.CompanionsCount, rsvp.Message, rsvp.RespondedAt,
		rsvp.CreatedAt, rsvp.UpdatedAt,
	)
	return err
}

func (r *RSVPRepository) ListByEventID(ctx context.Context, eventID string) ([]*models.RSVP, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, event_id, guest_id, status, companions_count, message, responded_at, created_at, updated_at
		 FROM rsvps WHERE event_id = $1 ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rsvps []*models.RSVP
	for rows.Next() {
		var rv models.RSVP
		if err := rows.Scan(
			&rv.ID, &rv.EventID, &rv.GuestID, &rv.Status,
			&rv.CompanionsCount, &rv.Message, &rv.RespondedAt,
			&rv.CreatedAt, &rv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rsvps = append(rsvps, &rv)
	}
	return rsvps, rows.Err()
}
