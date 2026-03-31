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
	logoURL     string
	senderEmail string
	senderName  string
}

type BrevoSenderOptions struct {
	APIKey      string
	AppName     string
	HTTPClient  *http.Client
	LogoURL     string
	SenderEmail string
	SenderName  string
}

func NewBrevoSender(options BrevoSenderOptions) *BrevoSender {
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	appName := strings.TrimSpace(options.AppName)
	if appName == "" {
		appName = "MyEvent"
	}

	return &BrevoSender{
		apiKey:      strings.TrimSpace(options.APIKey),
		appName:     appName,
		httpClient:  httpClient,
		logoURL:     strings.TrimSpace(options.LogoURL),
		senderEmail: strings.TrimSpace(options.SenderEmail),
		senderName:  strings.TrimSpace(options.SenderName),
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
	logoBlock := ""
	if s.logoURL != "" {
		logoURL := html.EscapeString(s.logoURL)
		logoBlock = fmt.Sprintf(
			`<img src="%s" alt="%s" width="48" height="48" style="display:block;width:48px;height:48px;border-radius:14px;border:0;outline:none;text-decoration:none">`,
			logoURL,
			html.EscapeString(s.appName),
		)
	}

	return fmt.Sprintf(`
<div style="margin:0;padding:32px 16px;background:#f8fafc;font-family:Arial,sans-serif;color:#0f172a">
  <div style="max-width:560px;margin:0 auto">
    <div style="padding:0 0 16px 0;text-align:center">
      %s
      <div style="margin-top:12px;font-size:28px;line-height:1;font-weight:700;letter-spacing:-0.02em;color:#0f172a">
        <span style="color:#7c3aed;font-weight:300;text-transform:uppercase;letter-spacing:0.18em;font-size:14px;vertical-align:middle">my</span>
        <span style="margin-left:6px;vertical-align:middle">event</span>
      </div>
    </div>

    <div style="background:linear-gradient(180deg,#ffffff 0%%,#faf5ff 100%%);border:1px solid #e9d5ff;border-radius:24px;padding:32px;box-shadow:0 10px 30px rgba(15,23,42,0.08)">
      <div style="display:inline-block;padding:6px 12px;border-radius:999px;background:#f3e8ff;color:#6d28d9;font-size:12px;font-weight:700;letter-spacing:0.12em;text-transform:uppercase">
        Recuperacao de senha
      </div>

      <h1 style="margin:18px 0 12px 0;font-size:28px;line-height:1.2;color:#0f172a">
        Ola, %s
      </h1>

      <p style="margin:0 0 16px 0;font-size:16px;line-height:1.7;color:#475569">
        Recebemos um pedido para redefinir a senha da sua conta no %s.
      </p>

      <p style="margin:0 0 24px 0;font-size:16px;line-height:1.7;color:#475569">
        Para continuar, clique no botao abaixo. Esse link expira em <strong style="color:#0f172a">%s</strong>.
      </p>

      <div style="margin:0 0 24px 0">
        <a href="%s" style="display:inline-block;background:#7c3aed;color:#ffffff;text-decoration:none;padding:14px 22px;border-radius:14px;font-size:15px;font-weight:700">
          Redefinir senha
        </a>
      </div>

      <div style="margin:0 0 24px 0;padding:16px 18px;border-radius:16px;background:#ffffff;border:1px solid #e2e8f0">
        <p style="margin:0 0 8px 0;font-size:13px;font-weight:700;color:#334155;text-transform:uppercase;letter-spacing:0.08em">
          Link direto
        </p>
        <p style="margin:0;font-size:14px;line-height:1.7;word-break:break-all">
          <a href="%s" style="color:#7c3aed;text-decoration:none">%s</a>
        </p>
      </div>

      <p style="margin:0;font-size:14px;line-height:1.7;color:#64748b">
        Se voce nao solicitou a recuperacao, pode ignorar este e-mail com seguranca.
      </p>
    </div>

    <p style="margin:18px 0 0 0;text-align:center;font-size:12px;line-height:1.6;color:#94a3b8">
      Enviado por %s
    </p>
  </div>
</div>`, logoBlock, name, html.EscapeString(s.appName), expiresIn, resetURL, resetURL, resetURL, html.EscapeString(s.appName))
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
