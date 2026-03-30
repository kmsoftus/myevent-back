package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type EventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{db: db}
}

const eventColumns = `id, user_id, title, slug, type, description, date, time,
	location_name, address, cover_image_url, host_message,
	theme, primary_color, secondary_color, background_color, text_color,
	pix_key, pix_holder_name, status, created_at, updated_at`

func scanEvent(row pgx.Row) (*models.Event, error) {
	var e models.Event
	err := row.Scan(
		&e.ID, &e.UserID, &e.Title, &e.Slug, &e.Type, &e.Description,
		&e.Date, &e.Time, &e.LocationName, &e.Address, &e.CoverImageURL,
		&e.HostMessage, &e.Theme, &e.PrimaryColor, &e.SecondaryColor,
		&e.BackgroundColor, &e.TextColor, &e.PixKey, &e.PixHolderName,
		&e.Status, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepository) Create(ctx context.Context, event *models.Event) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO events (id, user_id, title, slug, type, description, date, time,
			location_name, address, cover_image_url, host_message,
			theme, primary_color, secondary_color, background_color, text_color,
			pix_key, pix_holder_name, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
		event.ID, event.UserID, event.Title, event.Slug, event.Type, event.Description,
		event.Date, event.Time, event.LocationName, event.Address, event.CoverImageURL,
		event.HostMessage, event.Theme, event.PrimaryColor, event.SecondaryColor,
		event.BackgroundColor, event.TextColor, event.PixKey, event.PixHolderName,
		event.Status, event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return err
	}
	return nil
}

func (r *EventRepository) ListByUserID(ctx context.Context, userID string) ([]*models.Event, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+eventColumns+` FROM events WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	row := r.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id)
	e, err := scanEvent(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *EventRepository) GetBySlug(ctx context.Context, slug string) (*models.Event, error) {
	row := r.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE slug = $1`, slug)
	e, err := scanEvent(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *EventRepository) Update(ctx context.Context, event *models.Event) error {
	_, err := r.db.Exec(ctx,
		`UPDATE events SET title=$1, slug=$2, type=$3, description=$4, date=$5, time=$6,
			location_name=$7, address=$8, cover_image_url=$9, host_message=$10,
			theme=$11, primary_color=$12, secondary_color=$13, background_color=$14, text_color=$15,
			pix_key=$16, pix_holder_name=$17, status=$18, updated_at=$19
		 WHERE id=$20`,
		event.Title, event.Slug, event.Type, event.Description, event.Date, event.Time,
		event.LocationName, event.Address, event.CoverImageURL, event.HostMessage,
		event.Theme, event.PrimaryColor, event.SecondaryColor, event.BackgroundColor, event.TextColor,
		event.PixKey, event.PixHolderName, event.Status, event.UpdatedAt, event.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return err
	}
	return nil
}

func (r *EventRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM events WHERE id = $1`, id)
	return err
}
