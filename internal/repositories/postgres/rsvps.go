package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type RSVPRepository struct {
	db *pgxpool.Pool
}

func NewRSVPRepository(db *pgxpool.Pool) *RSVPRepository {
	return &RSVPRepository{db: db}
}

func (r *RSVPRepository) Upsert(ctx context.Context, rsvp *models.RSVP) error {
	names := rsvp.CompanionNames
	if names == nil {
		names = []string{}
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO rsvps (id, event_id, guest_id, status, companions_count, companion_names, message, responded_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (guest_id) DO UPDATE SET
			status = EXCLUDED.status,
			companions_count = EXCLUDED.companions_count,
			companion_names = EXCLUDED.companion_names,
			message = EXCLUDED.message,
			responded_at = EXCLUDED.responded_at,
			updated_at = EXCLUDED.updated_at`,
		rsvp.ID, rsvp.EventID, rsvp.GuestID, rsvp.Status,
		rsvp.CompanionsCount, names, rsvp.Message, rsvp.RespondedAt,
		rsvp.CreatedAt, rsvp.UpdatedAt,
	)
	return err
}

func (r *RSVPRepository) GetByGuestID(ctx context.Context, guestID string) (*models.RSVP, error) {
	var rv models.RSVP
	var nameArray pgtype.Array[string]
	err := r.db.QueryRow(ctx,
		`SELECT id, event_id, guest_id, status, companions_count, companion_names, message, responded_at, created_at, updated_at
		 FROM rsvps WHERE guest_id = $1`, guestID,
	).Scan(
		&rv.ID, &rv.EventID, &rv.GuestID, &rv.Status,
		&rv.CompanionsCount, &nameArray, &rv.Message, &rv.RespondedAt,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	if nameArray.Valid {
		rv.CompanionNames = nameArray.Elements
	} else {
		rv.CompanionNames = []string{}
	}
	return &rv, nil
}

func (r *RSVPRepository) ListByEventID(ctx context.Context, eventID string) ([]*models.RSVP, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, event_id, guest_id, status, companions_count, companion_names, message, responded_at, created_at, updated_at
		 FROM rsvps WHERE event_id = $1 ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rsvps []*models.RSVP
	for rows.Next() {
		var rv models.RSVP
		var nameArray pgtype.Array[string]
		if err := rows.Scan(
			&rv.ID, &rv.EventID, &rv.GuestID, &rv.Status,
			&rv.CompanionsCount, &nameArray, &rv.Message, &rv.RespondedAt,
			&rv.CreatedAt, &rv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if nameArray.Valid {
			rv.CompanionNames = nameArray.Elements
		} else {
			rv.CompanionNames = []string{}
		}
		rsvps = append(rsvps, &rv)
	}
	return rsvps, rows.Err()
}

func (r *RSVPRepository) CountByEventID(ctx context.Context, eventID string) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM rsvps WHERE event_id = $1`, eventID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *RSVPRepository) ListByEventIDPaged(ctx context.Context, eventID string, limit, offset int) ([]*models.RSVP, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, event_id, guest_id, status, companions_count, companion_names, message, responded_at, created_at, updated_at
		 FROM rsvps
		 WHERE event_id = $1
		 ORDER BY created_at ASC
		 LIMIT $2 OFFSET $3`,
		eventID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rsvps []*models.RSVP
	for rows.Next() {
		var rv models.RSVP
		var nameArray pgtype.Array[string]
		if err := rows.Scan(
			&rv.ID, &rv.EventID, &rv.GuestID, &rv.Status,
			&rv.CompanionsCount, &nameArray, &rv.Message, &rv.RespondedAt,
			&rv.CreatedAt, &rv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if nameArray.Valid {
			rv.CompanionNames = nameArray.Elements
		} else {
			rv.CompanionNames = []string{}
		}
		rsvps = append(rsvps, &rv)
	}
	return rsvps, rows.Err()
}
