package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"myevent-back/internal/auth"
	"myevent-back/internal/config"
	"myevent-back/internal/database"
	"myevent-back/internal/http/routes"
	"myevent-back/internal/repositories/postgres"
	"myevent-back/internal/services"
	"myevent-back/internal/storage"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
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
	events := postgres.NewEventRepository(db)
	guests := postgres.NewGuestRepository(db)
	rsvps := postgres.NewRSVPRepository(db)
	gifts := postgres.NewGiftRepository(db)
	giftTransactions := postgres.NewGiftTransactionRepository(db)

	authService := services.NewAuthService(users, jwtManager)
	eventService := services.NewEventService(events)
	guestService := services.NewGuestService(events, guests)
	rsvpService := services.NewRSVPService(events, guests, rsvps)
	checkInService := services.NewCheckInService(events, guests)
	giftService := services.NewGiftService(events, gifts)
	giftTransactionService := services.NewGiftTransactionService(events, gifts, giftTransactions)
	dashboardService := services.NewDashboardService(events, guests, gifts)

	objectStorage, err := buildStorage(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	uploadService := services.NewUploadService(objectStorage, cfg.UploadMaxSizeBytes)

	router := routes.NewRouter(
		cfg,
		jwtManager,
		authService,
		eventService,
		guestService,
		rsvpService,
		checkInService,
		giftService,
		giftTransactionService,
		dashboardService,
		uploadService,
	)

	log.Printf("myevent-back listening on :%s", cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, router); err != nil {
		log.Fatal(err)
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
