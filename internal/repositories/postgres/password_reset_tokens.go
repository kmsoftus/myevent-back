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

type PasswordResetTokenRepository struct {
	db *pgxpool.Pool
}

func NewPasswordResetTokenRepository(db *pgxpool.Pool) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

func (r *PasswordResetTokenRepository) Create(ctx context.Context, token *models.PasswordResetToken) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.UsedAt, token.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return err
	}

	return nil
}

func (r *PasswordResetTokenRepository) DeleteActiveByUserID(ctx context.Context, userID string, now time.Time) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM password_reset_tokens
		 WHERE user_id = $1
		   AND used_at IS NULL
		   AND expires_at > $2`,
		userID, now,
	)
	return err
}

func (r *PasswordResetTokenRepository) Consume(ctx context.Context, tokenHash string, now time.Time) (*models.PasswordResetToken, error) {
	var token models.PasswordResetToken

	err := r.db.QueryRow(ctx,
		`UPDATE password_reset_tokens
		    SET used_at = $2
		  WHERE token_hash = $1
		    AND used_at IS NULL
		    AND expires_at > $2
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at`,
		tokenHash, now,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	return &token, nil
}
