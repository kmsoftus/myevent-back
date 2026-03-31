package mailer

import (
	"context"
	"time"
)

type PasswordResetMessage struct {
	ExpiresIn time.Duration
	ResetURL  string
	ToEmail   string
	ToName    string
}

type PasswordResetSender interface {
	SendPasswordReset(ctx context.Context, message PasswordResetMessage) error
}

type NoopSender struct{}

func (NoopSender) SendPasswordReset(_ context.Context, _ PasswordResetMessage) error {
	return nil
}
