package routes

import (
	"net/http"
	"testing"
	"time"

	"myevent-back/internal/models"
)

func TestLoginRateLimitReturns429(t *testing.T) {
	router := newTestRouter(t)

	for range 5 {
		response := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
			"email":    "naoexiste@example.com",
			"password": "senha-incorreta",
		})
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected login attempt to return 401 before rate limit, got %d", response.Code)
		}
	}

	limitedResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    "naoexiste@example.com",
		"password": "senha-incorreta",
	})
	if limitedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding login rate limit, got %d", limitedResponse.Code)
	}
	if limitedResponse.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on rate limited response")
	}
}

func TestPublicRSVPRateLimitReturns429(t *testing.T) {
	router, _, _, store, _ := newTestRouterWithDeps(t)

	now := time.Now().UTC()
	event := &models.Event{
		ID:        "event-rate-limit",
		UserID:    "user-rate-limit",
		Title:     "Open RSVP",
		Slug:      "open-rsvp",
		Status:    "published",
		OpenRSVP:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Events().Create(t.Context(), event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	for range 10 {
		response := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+event.Slug+"/rsvp", "", map[string]any{
			"guest_identifier": "Convidado da taxa",
			"status":           "confirmed",
		})
		if response.Code != http.StatusOK {
			t.Fatalf("expected RSVP attempt to return 200 before rate limit, got %d", response.Code)
		}
	}

	limitedResponse := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+event.Slug+"/rsvp", "", map[string]any{
		"guest_identifier": "Convidado da taxa",
		"status":           "confirmed",
	})
	if limitedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding RSVP rate limit, got %d", limitedResponse.Code)
	}
}
