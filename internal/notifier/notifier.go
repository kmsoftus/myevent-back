package notifier

import (
	"context"
	"time"

	"myevent-back/internal/models"
)

type NewRegistrationMessage struct {
	UserID       string
	Name         string
	Email        string
	ContactPhone string
	Attribution  models.UserAttribution
	CreatedAt    time.Time
}

type RegistrationSender interface {
	SendNewRegistration(ctx context.Context, message NewRegistrationMessage) error
}

type NoopRegistrationSender struct{}

func (NoopRegistrationSender) SendNewRegistration(_ context.Context, _ NewRegistrationMessage) error {
	return nil
}
