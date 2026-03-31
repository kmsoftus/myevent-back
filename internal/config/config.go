package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	AppPort            string
	AppBaseURL         string
	FrontendURL        string
	EmailLogoURL       string
	PasswordResetURL   string
	TelegramBotToken   string
	TelegramGroupID    string
	JWTSecret          string
	JWTExpiresIn       time.Duration
	DatabaseURL        string
	CORSAllowedOrigins []string
	BrevoAPIKey        string
	BrevoSenderEmail   string
	BrevoSenderName    string
	PasswordResetTTL   time.Duration
	R2AccountID        string
	R2AccessKeyID      string
	R2SecretAccessKey  string
	R2Bucket           string
	R2Region           string
	R2Endpoint         string
	R2PublicURL        string
	LocalUploadDir     string
	UploadMaxSizeBytes int64
}

func Load() Config {
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")

	return Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		AppPort:            getEnv("APP_PORT", "8080"),
		AppBaseURL:         getEnv("APP_BASE_URL", "http://localhost:8080"),
		FrontendURL:        frontendURL,
		EmailLogoURL:       getEnv("EMAIL_LOGO_URL", strings.TrimRight(frontendURL, "/")+"/brand/myevent-social-avatar.png"),
		PasswordResetURL:   getEnv("PASSWORD_RESET_URL", strings.TrimRight(frontendURL, "/")+"/redefinir-senha"),
		TelegramBotToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramGroupID:    getEnv("TELEGRAM_GROUP_ID", ""),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret"),
		JWTExpiresIn:       getDurationEnv("JWT_EXPIRES_IN", 168*time.Hour),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		BrevoAPIKey:        getEnv("BREVO_API_KEY", ""),
		BrevoSenderEmail:   getEnv("BREVO_SENDER_EMAIL", ""),
		BrevoSenderName:    getEnv("BREVO_SENDER_NAME", "MyEvent"),
		PasswordResetTTL:   getDurationEnv("PASSWORD_RESET_TTL", time.Hour),
		R2AccountID:        getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:      getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:           getEnv("R2_BUCKET", ""),
		R2Region:           getEnv("R2_REGION", "auto"),
		R2Endpoint:         getEnv("R2_ENDPOINT", ""),
		R2PublicURL:        getEnv("R2_PUBLIC_URL", ""),
		LocalUploadDir:     getEnv("LOCAL_UPLOAD_DIR", ".data/uploads"),
		UploadMaxSizeBytes: getInt64Env("UPLOAD_MAX_SIZE_BYTES", 10<<20),
	}
}

func (c Config) UseR2Storage() bool {
	return strings.TrimSpace(c.R2AccessKeyID) != "" &&
		strings.TrimSpace(c.R2SecretAccessKey) != "" &&
		strings.TrimSpace(c.R2Bucket) != "" &&
		strings.TrimSpace(c.R2Endpoint) != "" &&
		strings.TrimSpace(c.R2PublicURL) != ""
}

func (c Config) UseBrevoEmail() bool {
	return strings.TrimSpace(c.BrevoAPIKey) != "" &&
		strings.TrimSpace(c.BrevoSenderEmail) != ""
}

func (c Config) UseTelegramNotifications() bool {
	return strings.TrimSpace(c.TelegramBotToken) != "" &&
		strings.TrimSpace(c.TelegramGroupID) != ""
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}

	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getInt64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	if len(values) == 0 {
		return []string{"http://localhost:3000"}
	}

	return values
}
