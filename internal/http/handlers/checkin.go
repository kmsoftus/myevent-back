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

	result, err := h.service.CheckIn(r.Context(), userID, chi.URLParam(r, "eventId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewCheckInResponse(result.Guest, result.AlreadyCheckedIn, result.CompanionNames))
}

func (h *CheckInHandler) ListGuests(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	page, pageSize := apphttp.ReadPagination(r)
	guests, err := h.service.ListGuests(r.Context(), userID, chi.URLParam(r, "eventId"), page, pageSize)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GuestResponse, 0, len(guests.Items))
	for _, guest := range guests.Items {
		response = append(response, dto.NewGuestResponse(guest))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.PaginatedResponse[dto.GuestResponse]{
		Items:      response,
		Total:      guests.Total,
		Page:       guests.Page,
		PageSize:   guests.PageSize,
		TotalPages: guests.TotalPages,
	})
}
