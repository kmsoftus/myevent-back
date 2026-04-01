package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type RSVPHandler struct {
	service *services.RSVPService
}

func NewRSVPHandler(service *services.RSVPService) *RSVPHandler {
	return &RSVPHandler{service: service}
}

func (h *RSVPHandler) SearchPublic(w nethttp.ResponseWriter, r *nethttp.Request) {
	query := r.URL.Query().Get("q")
	candidates, err := h.service.SearchGuestsBySlug(r.Context(), chi.URLParam(r, "slug"), query)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}
	apphttp.WriteJSON(w, nethttp.StatusOK, candidates)
}

func (h *RSVPHandler) SubmitPublic(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.CreateRSVPRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	details, err := h.service.SubmitBySlug(r.Context(), chi.URLParam(r, "slug"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewRSVPResponse(details.RSVP, details.Guest))
}

func (h *RSVPHandler) ListByEvent(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	details, err := h.service.ListByEvent(r.Context(), userID, chi.URLParam(r, "eventId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.RSVPResponse, 0, len(details))
	for _, item := range details {
		response = append(response, dto.NewRSVPResponse(item.RSVP, item.Guest))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}
