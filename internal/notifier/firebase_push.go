package notifier

import (
	"context"
	"encoding/json"
	"os"
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
	ProjectID       string
}

func NewFirebasePushSender(ctx context.Context, options FirebasePushSenderOptions) (*FirebasePushSender, error) {
	credentialsFile := strings.TrimSpace(options.CredentialsFile)
	credentialsJSON := normalizeCredentialsJSON(options.CredentialsJSON)
	projectID := strings.TrimSpace(options.ProjectID)

	var appOption option.ClientOption
	if credentialsJSON != "" {
		appOption = option.WithCredentialsJSON([]byte(credentialsJSON))
		if projectID == "" {
			projectID = extractProjectIDFromJSON(credentialsJSON)
		}
	} else {
		appOption = option.WithCredentialsFile(credentialsFile)
		if projectID == "" {
			projectID = extractProjectIDFromFile(credentialsFile)
		}
	}

	firebaseConfig := &firebase.Config{}
	if projectID != "" {
		firebaseConfig.ProjectID = projectID
	}

	app, err := firebase.NewApp(ctx, firebaseConfig, appOption)
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

func normalizeCredentialsJSON(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}

	var quoted string
	if err := json.Unmarshal([]byte(raw), &quoted); err == nil {
		unquoted := strings.TrimSpace(quoted)
		if strings.HasPrefix(unquoted, "{") && strings.HasSuffix(unquoted, "}") {
			return unquoted
		}
	}

	return raw
}

func extractProjectIDFromJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}

	projectID, _ := payload["project_id"].(string)
	return strings.TrimSpace(projectID)
}

func extractProjectIDFromFile(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return extractProjectIDFromJSON(string(content))
}
