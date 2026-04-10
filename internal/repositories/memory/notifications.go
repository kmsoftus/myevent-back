package memory

import (
	"context"
	"sort"
	"time"

	"myevent-back/internal/models"
)

type notificationRepository struct {
	store *Store
}

func (r *notificationRepository) Create(ctx context.Context, n *models.Notification) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	cp := *n
	r.store.notifications[n.ID] = &cp
	return nil
}

func (r *notificationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Notification, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	var list []*models.Notification
	for _, n := range r.store.notifications {
		if n.UserID == userID {
			cp := *n
			list = append(list, &cp)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if offset >= len(list) {
		return nil, nil
	}
	list = list[offset:]
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (r *notificationRepository) CountUnreadByUserID(ctx context.Context, userID string) (int, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	count := 0
	for _, n := range r.store.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, id, userID string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	n, ok := r.store.notifications[id]
	if !ok || n.UserID != userID {
		return nil
	}
	if n.ReadAt == nil {
		now := time.Now().UTC()
		n.ReadAt = &now
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	now := time.Now().UTC()
	for _, n := range r.store.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			n.ReadAt = &now
		}
	}
	return nil
}
