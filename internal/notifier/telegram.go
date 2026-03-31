package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

const telegramAPIBaseURL = "https://api.telegram.org"

type TelegramSender struct {
	appName    string
	apiBaseURL string
	botToken   string
	chatID     string
	httpClient *http.Client
}

type TelegramSenderOptions struct {
	AppName    string
	APIBaseURL string
	BotToken   string
	ChatID     string
	HTTPClient *http.Client
}

func NewTelegramSender(options TelegramSenderOptions) *TelegramSender {
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	appName := strings.TrimSpace(options.AppName)
	if appName == "" {
		appName = "MyEvent"
	}

	apiBaseURL := strings.TrimRight(strings.TrimSpace(options.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = telegramAPIBaseURL
	}

	return &TelegramSender{
		appName:    appName,
		apiBaseURL: apiBaseURL,
		botToken:   strings.TrimSpace(options.BotToken),
		chatID:     strings.TrimSpace(options.ChatID),
		httpClient: httpClient,
	}
}

func (s *TelegramSender) SendNewRegistration(ctx context.Context, message NewRegistrationMessage) error {
	payload := map[string]any{
		"chat_id":                  s.chatID,
		"disable_web_page_preview": true,
		"parse_mode":               "HTML",
		"text":                     s.newRegistrationText(message),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.apiBaseURL+"/bot"+s.botToken+"/sendMessage",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("telegram responded with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func (s *TelegramSender) newRegistrationText(message NewRegistrationMessage) string {
	name := html.EscapeString(strings.TrimSpace(message.Name))
	if name == "" {
		name = "Sem nome"
	}

	email := html.EscapeString(strings.TrimSpace(message.Email))
	if email == "" {
		email = "Sem e-mail"
	}

	contactPhone := html.EscapeString(strings.TrimSpace(message.ContactPhone))

	createdAt := message.CreatedAt.UTC().Format("2006-01-02 15:04:05 MST")
	if message.CreatedAt.IsZero() {
		createdAt = time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	}

	lines := []string{
		fmt.Sprintf("Novo cadastro no <b>%s</b>", html.EscapeString(s.appName)),
		"",
		fmt.Sprintf("Nome: %s", name),
		fmt.Sprintf("E-mail: %s", email),
		fmt.Sprintf("User ID: <code>%s</code>", html.EscapeString(strings.TrimSpace(message.UserID))),
		fmt.Sprintf("Criado em: %s", html.EscapeString(createdAt)),
	}

	if contactPhone != "" {
		lines = append(lines, fmt.Sprintf("Contato: %s", contactPhone))
	}

	if source := html.EscapeString(strings.TrimSpace(message.Attribution.UTMSource)); source != "" {
		lines = append(lines, fmt.Sprintf("UTM Source: %s", source))
	}
	if medium := html.EscapeString(strings.TrimSpace(message.Attribution.UTMMedium)); medium != "" {
		lines = append(lines, fmt.Sprintf("UTM Medium: %s", medium))
	}
	if campaign := html.EscapeString(strings.TrimSpace(message.Attribution.UTMCampaign)); campaign != "" {
		lines = append(lines, fmt.Sprintf("UTM Campaign: %s", campaign))
	}
	if term := html.EscapeString(strings.TrimSpace(message.Attribution.UTMTerm)); term != "" {
		lines = append(lines, fmt.Sprintf("UTM Term: %s", term))
	}
	if content := html.EscapeString(strings.TrimSpace(message.Attribution.UTMContent)); content != "" {
		lines = append(lines, fmt.Sprintf("UTM Content: %s", content))
	}

	return strings.Join(lines, "\n")
}
