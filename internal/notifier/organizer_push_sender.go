package notifier

import "context"

type OrganizerPushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

type OrganizerPushSender interface {
	SendToDevice(ctx context.Context, deviceToken string, message OrganizerPushMessage) error
}

type NoopOrganizerPushSender struct{}

func (NoopOrganizerPushSender) SendToDevice(_ context.Context, _ string, _ OrganizerPushMessage) error {
	return nil
}
