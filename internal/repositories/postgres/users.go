package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

const userColumns = `id, name, email, contact_phone, accepted_terms, marketing_opt_in, password_hash, utm_source, utm_medium, utm_campaign, utm_term, utm_content, created_at, updated_at`

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, name, email, contact_phone, accepted_terms, marketing_opt_in, password_hash, utm_source, utm_medium, utm_campaign, utm_term, utm_content, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		user.ID,
		user.Name,
		user.Email,
		user.ContactPhone,
		user.AcceptedTerms,
		user.MarketingOptIn,
		user.PasswordHash,
		user.Attribution.UTMSource,
		user.Attribution.UTMMedium,
		user.Attribution.UTMCampaign,
		user.Attribution.UTMTerm,
		user.Attribution.UTMContent,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id,
	).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.ContactPhone,
		&u.AcceptedTerms,
		&u.MarketingOptIn,
		&u.PasswordHash,
		&u.Attribution.UTMSource,
		&u.Attribution.UTMMedium,
		&u.Attribution.UTMCampaign,
		&u.Attribution.UTMTerm,
		&u.Attribution.UTMContent,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email,
	).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.ContactPhone,
		&u.AcceptedTerms,
		&u.MarketingOptIn,
		&u.PasswordHash,
		&u.Attribution.UTMSource,
		&u.Attribution.UTMMedium,
		&u.Attribution.UTMCampaign,
		&u.Attribution.UTMTerm,
		&u.Attribution.UTMContent,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id, name, contactPhone string, updatedAt time.Time) error {
	commandTag, err := r.db.Exec(ctx,
		`UPDATE users
		    SET name = $2,
		        contact_phone = $3,
		        updated_at = $4
		  WHERE id = $1`,
		id, name, contactPhone, updatedAt,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error {
	commandTag, err := r.db.Exec(ctx,
		`UPDATE users
		    SET password_hash = $2,
		        updated_at = $3
		  WHERE id = $1`,
		id, passwordHash, updatedAt,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}

	return nil
}
