package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

func (r *GuestRepository) FindOrCreateOpenRSVPGuest(ctx context.Context, guest *models.Guest) (*models.Guest, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var eventID string
	if err := tx.QueryRow(ctx, `SELECT id FROM events WHERE id = $1 FOR UPDATE`, guest.EventID).Scan(&eventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	existing, err := scanGuest(tx.QueryRow(ctx,
		`SELECT `+guestColumns+` FROM guests
		 WHERE event_id = $1 AND LOWER(name) = LOWER($2)
		 ORDER BY created_at ASC
		 LIMIT 1`,
		guest.EventID, guest.Name,
	))
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO guests (id, event_id, name, email, phone, invite_code, short_code, qr_code_token,
			max_companions, rsvp_status, notes, checked_in_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		guest.ID, guest.EventID, guest.Name, guest.Email, guest.Phone,
		guest.InviteCode, guest.ShortCode, guest.QRCodeToken, guest.MaxCompanions,
		guest.RSVPStatus, guest.Notes, guest.CheckedInAt, guest.CreatedAt, guest.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, repositories.ErrConflict
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	copy := *guest
	return &copy, nil
}
