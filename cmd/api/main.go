package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"myevent-back/internal/auth"
	"myevent-back/internal/config"
	"myevent-back/internal/database"
	"myevent-back/internal/http/routes"
	"myevent-back/internal/mailer"
	"myevent-back/internal/notifier"
	"myevent-back/internal/repositories/postgres"
	"myevent-back/internal/services"
	"myevent-back/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	ctx := context.Background()

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}
	defer db.Close()

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiresIn)

	users := postgres.NewUserRepository(db)
	passwordResetTokens := postgres.NewPasswordResetTokenRepository(db)
	events := postgres.NewEventRepository(db)
	guests := postgres.NewGuestRepository(db)
	rsvps := postgres.NewRSVPRepository(db)
	gifts := postgres.NewGiftRepository(db)
	giftTransactions := postgres.NewGiftTransactionRepository(db)

	passwordResetSender := buildPasswordResetSender(cfg)
	registrationSender := buildRegistrationSender(cfg)
	authService := services.NewAuthService(
		users,
		passwordResetTokens,
		jwtManager,
		cfg.PasswordResetTTL,
		cfg.PasswordResetURL,
		passwordResetSender,
		registrationSender,
	)
	eventService := services.NewEventService(events)
	guestService := services.NewGuestService(events, guests)
	rsvpService := services.NewRSVPService(events, guests, rsvps, cfg.OpenRSVPDefaultMaxCompanions)
	checkInService := services.NewCheckInService(events, guests, rsvps)
	giftService := services.NewGiftService(events, gifts)
	giftTransactionService := services.NewGiftTransactionService(events, gifts, giftTransactions, cfg.GiftReservationTTL)
	dashboardService := services.NewDashboardService(events, guests, gifts)

	objectStorage, err := buildStorage(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	uploadService := services.NewUploadService(objectStorage, cfg.UploadMaxSizeBytes)
	accountService := services.NewAccountService(users, events, gifts, uploadService)

	router := routes.NewRouter(
		cfg,
		db,
		objectStorage,
		jwtManager,
		authService,
		accountService,
		eventService,
		guestService,
		rsvpService,
		checkInService,
		giftService,
		giftTransactionService,
		dashboardService,
		uploadService,
	)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.GiftReservationTTL > 0 && cfg.GiftSweepInterval > 0 {
		go startGiftPendingSweeper(serverCtx, giftTransactionService, cfg.GiftSweepInterval)
	}

	log.Printf("myevent-back listening on :%s", cfg.AppPort)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-serverCtx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("http server shutdown error: %v", err)
	}
}

func startGiftPendingSweeper(ctx context.Context, service *services.GiftTransactionService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := service.ExpirePending(context.Background())
			if err != nil {
				log.Printf("gift pending sweeper failed: %v", err)
				continue
			}
			if expired > 0 {
				log.Printf("gift pending sweeper expired %d transaction(s)", expired)
			}
		}
	}
}

func buildStorage(ctx context.Context, cfg config.Config) (storage.Provider, error) {
	if cfg.UseR2Storage() {
		return storage.NewR2Storage(ctx, storage.R2Config{
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretAccessKey,
			Bucket:          cfg.R2Bucket,
			Region:          defaultStorageRegion(cfg.R2Region),
			Endpoint:        cfg.R2Endpoint,
			PublicURL:       cfg.R2PublicURL,
		})
	}

	return storage.NewLocalStorage(cfg.LocalUploadDir, strings.TrimRight(cfg.AppBaseURL, "/")+"/uploads")
}

func defaultStorageRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return "auto"
	}
	return region
}

func buildPasswordResetSender(cfg config.Config) mailer.PasswordResetSender {
	if !cfg.UseBrevoEmail() {
		log.Print("password reset emails disabled: configure BREVO_API_KEY and BREVO_SENDER_EMAIL to enable Brevo")
		return mailer.NoopSender{}
	}

	return mailer.NewBrevoSender(mailer.BrevoSenderOptions{
		APIKey:      cfg.BrevoAPIKey,
		AppName:     "MyEvent",
		LogoURL:     cfg.EmailLogoURL,
		SenderEmail: cfg.BrevoSenderEmail,
		SenderName:  cfg.BrevoSenderName,
	})
}

func buildRegistrationSender(cfg config.Config) notifier.RegistrationSender {
	if !cfg.UseTelegramNotifications() {
		log.Print("telegram registration notifications disabled: configure TELEGRAM_BOT_TOKEN and TELEGRAM_GROUP_ID to enable")
		return notifier.NoopRegistrationSender{}
	}

	return notifier.NewTelegramSender(notifier.TelegramSenderOptions{
		AppName:  "MyEvent",
		BotToken: cfg.TelegramBotToken,
		ChatID:   cfg.TelegramGroupID,
	})
}
