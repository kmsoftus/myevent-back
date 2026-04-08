package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
)

type PushDeviceTokenRepository struct {
	db *pgxpool.Pool
}

func NewPushDeviceTokenRepository(db *pgxpool.Pool) *PushDeviceTokenRepository {
	return &PushDeviceTokenRepository{db: db}
}

func (r *PushDeviceTokenRepository) Upsert(ctx context.Context, token *models.PushDeviceToken) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO push_device_tokens (id, user_id, token, platform, created_at, updated_at, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			updated_at = EXCLUDED.updated_at,
			last_seen_at = EXCLUDED.last_seen_at`,
		token.ID,
		token.UserID,
		token.Token,
		token.Platform,
		token.CreatedAt,
		token.UpdatedAt,
		token.LastSeenAt,
	)
	return err
}

func (r *PushDeviceTokenRepository) ListByUserID(ctx context.Context, userID string) ([]*models.PushDeviceToken, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, token, platform, created_at, updated_at, last_seen_at
		 FROM push_device_tokens
		 WHERE user_id = $1
		 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.PushDeviceToken
	for rows.Next() {
		var token models.PushDeviceToken
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Token,
			&token.Platform,
			&token.CreatedAt,
			&token.UpdatedAt,
			&token.LastSeenAt,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, &token)
	}

	return tokens, rows.Err()
}

func (r *PushDeviceTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM push_device_tokens WHERE token = $1`, token)
	return err
}

func (r *PushDeviceTokenRepository) ListAll(ctx context.Context) ([]*models.PushDeviceToken, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, token, platform, created_at, updated_at, last_seen_at
		 FROM push_device_tokens
		 ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.PushDeviceToken
	for rows.Next() {
		var token models.PushDeviceToken
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Token,
			&token.Platform,
			&token.CreatedAt,
			&token.UpdatedAt,
			&token.LastSeenAt,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, &token)
	}

	return tokens, rows.Err()
}
