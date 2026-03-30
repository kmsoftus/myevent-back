package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"myevent-back/internal/auth"
	"myevent-back/internal/config"
	"myevent-back/internal/http/routes"
	"myevent-back/internal/repositories/memory"
	"myevent-back/internal/services"
	"myevent-back/internal/storage"
)

func main() {
	cfg := config.Load()
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiresIn)

	store := memory.NewStore()

	authService := services.NewAuthService(store.Users(), jwtManager)
	eventService := services.NewEventService(store.Events())
	guestService := services.NewGuestService(store.Events(), store.Guests())
	rsvpService := services.NewRSVPService(store.Events(), store.Guests(), store.RSVPs())
	checkInService := services.NewCheckInService(store.Events(), store.Guests())
	giftService := services.NewGiftService(store.Events(), store.Gifts())
	giftTransactionService := services.NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions())
	dashboardService := services.NewDashboardService(store.Events(), store.Guests(), store.Gifts())
	objectStorage, err := buildStorage(context.Background(), cfg)
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
