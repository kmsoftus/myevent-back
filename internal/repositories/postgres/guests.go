package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type GuestRepository struct {
	db *pgxpool.Pool
}

func NewGuestRepository(db *pgxpool.Pool) *GuestRepository {
	return &GuestRepository{db: db}
}

const guestColumns = `id, event_id, name, email, phone, invite_code, short_code, qr_code_token,
	max_companions, rsvp_status, checked_in_at, created_at, updated_at`

func scanGuest(row pgx.Row) (*models.Guest, error) {
	var g models.Guest
	err := row.Scan(
		&g.ID, &g.EventID, &g.Name, &g.Email, &g.Phone,
		&g.InviteCode, &g.ShortCode, &g.QRCodeToken, &g.MaxCompanions,
		&g.RSVPStatus, &g.CheckedInAt, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GuestRepository) Create(ctx context.Context, guest *models.Guest) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO guests (id, event_id, name, email, phone, invite_code, short_code, qr_code_token,
			max_companions, rsvp_status, checked_in_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		guest.ID, guest.EventID, guest.Name, guest.Email, guest.Phone,
		guest.InviteCode, guest.ShortCode, guest.QRCodeToken, guest.MaxCompanions,
		guest.RSVPStatus, guest.CheckedInAt, guest.CreatedAt, guest.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return err
	}
	return nil
}

func (r *GuestRepository) GetByShortCode(ctx context.Context, eventID, shortCode string) (*models.Guest, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+guestColumns+` FROM guests WHERE event_id = $1 AND short_code = $2`,
		eventID, shortCode,
	)
	g, err := scanGuest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *GuestRepository) SearchByName(ctx context.Context, eventID, query string, limit int) ([]*models.Guest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+guestColumns+` FROM guests
		 WHERE event_id = $1 AND name ILIKE '%' || $2 || '%'
		 ORDER BY name ASC
		 LIMIT $3`,
		eventID, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []*models.Guest
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}
	return guests, rows.Err()
}

func (r *GuestRepository) ListByEventID(ctx context.Context, eventID string) ([]*models.Guest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+guestColumns+` FROM guests WHERE event_id = $1 ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []*models.Guest
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}
	return guests, rows.Err()
}

func (r *GuestRepository) GetByID(ctx context.Context, id string) (*models.Guest, error) {
	row := r.db.QueryRow(ctx, `SELECT `+guestColumns+` FROM guests WHERE id = $1`, id)
	g, err := scanGuest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *GuestRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*models.Guest, error) {
	row := r.db.QueryRow(ctx, `SELECT `+guestColumns+` FROM guests WHERE invite_code = $1`, inviteCode)
	g, err := scanGuest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *GuestRepository) GetByQRCodeToken(ctx context.Context, qrCodeToken string) (*models.Guest, error) {
	row := r.db.QueryRow(ctx, `SELECT `+guestColumns+` FROM guests WHERE qr_code_token = $1`, qrCodeToken)
	g, err := scanGuest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *GuestRepository) Update(ctx context.Context, guest *models.Guest) error {
	_, err := r.db.Exec(ctx,
		`UPDATE guests SET name=$1, email=$2, phone=$3, max_companions=$4,
			rsvp_status=$5, checked_in_at=$6, short_code=$7, updated_at=$8
		 WHERE id=$9`,
		guest.Name, guest.Email, guest.Phone, guest.MaxCompanions,
		guest.RSVPStatus, guest.CheckedInAt, guest.ShortCode, guest.UpdatedAt, guest.ID,
	)
	return err
}

func (r *GuestRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM guests WHERE id = $1`, id)
	return err
}
