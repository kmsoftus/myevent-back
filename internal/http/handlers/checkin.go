package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type CheckInHandler struct {
	service *services.CheckInService
}

func NewCheckInHandler(service *services.CheckInService) *CheckInHandler {
	return &CheckInHandler{service: service}
}

func (h *CheckInHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.CheckInRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	guest, err := h.service.CheckIn(r.Context(), userID, chi.URLParam(r, "eventId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewCheckInResponse(guest))
}

func (h *CheckInHandler) ListGuests(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	guests, err := h.service.ListGuests(r.Context(), userID, chi.URLParam(r, "eventId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GuestResponse, 0, len(guests))
	for _, guest := range guests {
		response = append(response, dto.NewGuestResponse(guest))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}
