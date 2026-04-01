package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"myevent-back/internal/auth"
	"myevent-back/internal/config"
	"myevent-back/internal/http/handlers"
	authmiddleware "myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
	"myevent-back/internal/storage"
)

func NewRouter(
	cfg config.Config,
	db *pgxpool.Pool,
	objectStorage storage.Provider,
	jwtManager *auth.JWTManager,
	authService *services.AuthService,
	accountService *services.AccountService,
	eventService *services.EventService,
	guestService *services.GuestService,
	rsvpService *services.RSVPService,
	checkInService *services.CheckInService,
	giftService *services.GiftService,
	giftTransactionService *services.GiftTransactionService,
	dashboardService *services.DashboardService,
	uploadService *services.UploadService,
) http.Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.StripSlashes)
	router.Use(chimiddleware.Timeout(30 * time.Second))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	healthHandler := handlers.NewHealthHandler(db, objectStorage)
	authHandler := handlers.NewAuthHandler(authService, accountService)
	eventHandler := handlers.NewEventHandler(eventService)
	publicEventHandler := handlers.NewPublicEventHandler(eventService)
	guestHandler := handlers.NewGuestHandler(guestService)
	rsvpHandler := handlers.NewRSVPHandler(rsvpService)
	checkInHandler := handlers.NewCheckInHandler(checkInService)
	giftHandler := handlers.NewGiftHandler(giftService)
	giftTransactionHandler := handlers.NewGiftTransactionHandler(giftTransactionService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	uploadHandler := handlers.NewUploadHandler(uploadService)

	if !cfg.UseR2Storage() {
		router.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.LocalUploadDir))))
	}

	router.Get("/health", healthHandler.Check)

	router.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)

			r.Group(func(r chi.Router) {
				r.Use(authmiddleware.Authenticator(jwtManager))
				r.Get("/me", authHandler.Me)
				r.Patch("/me", authHandler.UpdateMe)
				r.Delete("/me", authHandler.DeleteMe)
			})
		})

		r.Route("/public", func(r chi.Router) {
			r.Get("/events/{slug}", publicEventHandler.GetBySlug)
			r.Get("/events/{slug}/rsvp/lookup", rsvpHandler.LookupPublic)
			r.Get("/events/{slug}/rsvp/search", rsvpHandler.SearchPublic)
			r.Post("/events/{slug}/rsvp", rsvpHandler.SubmitPublic)
			r.Get("/events/{slug}/gifts", giftHandler.ListPublic)
			r.Post("/events/{slug}/gifts/{giftId}/reserve", giftTransactionHandler.ReservePublic)
			r.Post("/events/{slug}/gifts/{giftId}/pix", giftTransactionHandler.PixPublic)
		})

		r.Group(func(r chi.Router) {
			r.Use(authmiddleware.Authenticator(jwtManager))

			r.Route("/uploads", func(r chi.Router) {
				r.Post("/", uploadHandler.Create)
				r.Delete("/", uploadHandler.Delete)
			})

			r.Route("/events", func(r chi.Router) {
				r.Post("/", eventHandler.Create)
				r.Get("/", eventHandler.List)
				r.Get("/{eventId}", eventHandler.GetByID)
				r.Patch("/{eventId}", eventHandler.Update)
				r.Patch("/{eventId}/status", eventHandler.UpdateStatus)
				r.Delete("/{eventId}", eventHandler.Delete)

				r.Route("/{eventId}/guests", func(r chi.Router) {
					r.Post("/", guestHandler.Create)
					r.Get("/", guestHandler.List)
					r.Get("/{guestId}", guestHandler.GetByID)
					r.Get("/{guestId}/qrcode", guestHandler.GetQRCode)
					r.Patch("/{guestId}", guestHandler.Update)
					r.Delete("/{guestId}", guestHandler.Delete)
				})

				r.Get("/{eventId}/rsvps", rsvpHandler.ListByEvent)
				r.Post("/{eventId}/checkin", checkInHandler.Create)
				r.Get("/{eventId}/checkin/guests", checkInHandler.ListGuests)
				r.Route("/{eventId}/gifts", func(r chi.Router) {
					r.Post("/", giftHandler.Create)
					r.Get("/", giftHandler.List)
					r.Get("/{giftId}", giftHandler.GetByID)
					r.Patch("/{giftId}", giftHandler.Update)
					r.Delete("/{giftId}", giftHandler.Delete)
				})
				r.Get("/{eventId}/gift-transactions", giftTransactionHandler.ListByEvent)
				r.Patch("/{eventId}/gift-transactions/{transactionId}/confirm", giftTransactionHandler.Confirm)
				r.Patch("/{eventId}/gift-transactions/{transactionId}/cancel", giftTransactionHandler.Cancel)
				r.Get("/{eventId}/dashboard", dashboardHandler.GetByEvent)
			})
		})
	})

	return router
}
