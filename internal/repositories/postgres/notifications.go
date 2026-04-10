package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/models"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, title, body, data, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		n.ID, n.UserID, n.Type, n.Title, n.Body, data, n.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Notification, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, type, title, body, data, read_at, created_at
		 FROM notifications
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Notification
	for rows.Next() {
		var n models.Notification
		var rawData []byte
		var readAt *time.Time

		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &rawData, &readAt, &n.CreatedAt); err != nil {
			return nil, err
		}

		n.ReadAt = readAt
		if err := json.Unmarshal(rawData, &n.Data); err != nil {
			n.Data = map[string]string{}
		}

		list = append(list, &n)
	}

	return list, rows.Err()
}

func (r *NotificationRepository) CountUnreadByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		userID,
	).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`,
		id, userID,
	)
	return err
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`,
		userID,
	)
	return err
}
