package mailer

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

const brevoSMTPURL = "https://api.brevo.com/v3/smtp/email"

type BrevoSender struct {
	apiKey      string
	appName     string
	httpClient  *http.Client
	senderEmail string
	senderName  string
}

func NewBrevoSender(apiKey, senderEmail, senderName string, httpClient *http.Client) *BrevoSender {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &BrevoSender{
		apiKey:      strings.TrimSpace(apiKey),
		appName:     "MyEvent",
		httpClient:  httpClient,
		senderEmail: strings.TrimSpace(senderEmail),
		senderName:  strings.TrimSpace(senderName),
	}
}

func (s *BrevoSender) SendPasswordReset(ctx context.Context, message PasswordResetMessage) error {
	payload := map[string]any{
		"sender": map[string]string{
			"email": s.senderEmail,
			"name":  s.senderName,
		},
		"to": []map[string]string{
			{
				"email": strings.TrimSpace(message.ToEmail),
				"name":  strings.TrimSpace(message.ToName),
			},
		},
		"subject":     "Recuperacao de senha - MyEvent",
		"htmlContent": s.passwordResetHTML(message),
		"textContent": s.passwordResetText(message),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal brevo payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, brevoSMTPURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create brevo request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("api-key", s.apiKey)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send email with brevo: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("brevo responded with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func (s *BrevoSender) passwordResetHTML(message PasswordResetMessage) string {
	name := html.EscapeString(strings.TrimSpace(message.ToName))
	if name == "" {
		name = "oi"
	}

	resetURL := html.EscapeString(message.ResetURL)
	expiresIn := html.EscapeString(formatDurationPTBR(message.ExpiresIn))

	return fmt.Sprintf(`
<div style="font-family:Arial,sans-serif;line-height:1.6;color:#111827">
  <p>Ola, %s.</p>
  <p>Recebemos um pedido para redefinir a senha da sua conta no %s.</p>
  <p>
    <a href="%s" style="display:inline-block;background:#111827;color:#ffffff;text-decoration:none;padding:12px 20px;border-radius:10px;font-weight:600">
      Redefinir senha
    </a>
  </p>
  <p>Se preferir, copie e cole este link no navegador:</p>
  <p><a href="%s">%s</a></p>
  <p>Esse link expira em %s.</p>
  <p>Se voce nao solicitou a recuperacao, pode ignorar este e-mail.</p>
</div>`, name, html.EscapeString(s.appName), resetURL, resetURL, resetURL, expiresIn)
}

func (s *BrevoSender) passwordResetText(message PasswordResetMessage) string {
	name := strings.TrimSpace(message.ToName)
	if name == "" {
		name = "oi"
	}

	return fmt.Sprintf(
		"Ola, %s.\n\nRecebemos um pedido para redefinir a senha da sua conta no %s.\n\nAbra este link para continuar:\n%s\n\nEsse link expira em %s.\n\nSe voce nao solicitou a recuperacao, pode ignorar este e-mail.\n",
		name,
		s.appName,
		message.ResetURL,
		formatDurationPTBR(message.ExpiresIn),
	)
}

func formatDurationPTBR(value time.Duration) string {
	if value <= 0 {
		return "alguns minutos"
	}

	if value%time.Hour == 0 {
		hours := int(value / time.Hour)
		if hours == 1 {
			return "1 hora"
		}
		return fmt.Sprintf("%d horas", hours)
	}

	if value%time.Minute == 0 {
		minutes := int(value / time.Minute)
		if minutes == 1 {
			return "1 minuto"
		}
		return fmt.Sprintf("%d minutos", minutes)
	}

	seconds := int(value / time.Second)
	if seconds == 1 {
		return "1 segundo"
	}

	return fmt.Sprintf("%d segundos", seconds)
}
