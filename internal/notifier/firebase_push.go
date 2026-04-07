package notifier

import (
	"context"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FirebasePushSender struct {
	client *messaging.Client
}

type FirebasePushSenderOptions struct {
	CredentialsFile string
	CredentialsJSON string
}

func NewFirebasePushSender(ctx context.Context, options FirebasePushSenderOptions) (*FirebasePushSender, error) {
	credentialsFile := strings.TrimSpace(options.CredentialsFile)
	credentialsJSON := strings.TrimSpace(options.CredentialsJSON)

	var appOption option.ClientOption
	if credentialsJSON != "" {
		appOption = option.WithCredentialsJSON([]byte(credentialsJSON))
	} else {
		appOption = option.WithCredentialsFile(credentialsFile)
	}

	app, err := firebase.NewApp(
		ctx,
		nil,
		appOption,
	)
	if err != nil {
		return nil, err
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebasePushSender{client: client}, nil
}

func (s *FirebasePushSender) SendToDevice(ctx context.Context, deviceToken string, message OrganizerPushMessage) error {
	payload := &messaging.Message{
		Token: strings.TrimSpace(deviceToken),
		Notification: &messaging.Notification{
			Title: strings.TrimSpace(message.Title),
			Body:  strings.TrimSpace(message.Body),
		},
		Data: message.Data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{"apns-priority": "10"},
		},
	}

	_, err := s.client.Send(ctx, payload)
	return err
}
