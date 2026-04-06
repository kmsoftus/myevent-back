package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type GalleryPhotoRepository struct {
	db *pgxpool.Pool
}

func NewGalleryPhotoRepository(db *pgxpool.Pool) *GalleryPhotoRepository {
	return &GalleryPhotoRepository{db: db}
}

const galleryPhotoColumns = `id, event_id, image_url, position, created_at`

func scanGalleryPhoto(row pgx.Row) (*models.GalleryPhoto, error) {
	var p models.GalleryPhoto
	err := row.Scan(&p.ID, &p.EventID, &p.ImageURL, &p.Position, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *GalleryPhotoRepository) Create(ctx context.Context, photo *models.GalleryPhoto) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO event_gallery_photos (id, event_id, image_url, position, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		photo.ID, photo.EventID, photo.ImageURL, photo.Position, photo.CreatedAt,
	)
	return err
}

func (r *GalleryPhotoRepository) ListByEventID(ctx context.Context, eventID string) ([]*models.GalleryPhoto, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+galleryPhotoColumns+` FROM event_gallery_photos WHERE event_id = $1 ORDER BY position ASC, created_at ASC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []*models.GalleryPhoto
	for rows.Next() {
		p, err := scanGalleryPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

func (r *GalleryPhotoRepository) CountByEventID(ctx context.Context, eventID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_gallery_photos WHERE event_id = $1`, eventID,
	).Scan(&count)
	return count, err
}

func (r *GalleryPhotoRepository) GetByID(ctx context.Context, id string) (*models.GalleryPhoto, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+galleryPhotoColumns+` FROM event_gallery_photos WHERE id = $1`, id,
	)
	p, err := scanGalleryPhoto(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *GalleryPhotoRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM event_gallery_photos WHERE id = $1`, id)
	return err
}
