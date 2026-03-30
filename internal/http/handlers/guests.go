package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type GuestHandler struct {
	service *services.GuestService
}

func NewGuestHandler(service *services.GuestService) *GuestHandler {
	return &GuestHandler{service: service}
}

func (h *GuestHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.CreateGuestRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	guest, err := h.service.Create(r.Context(), userID, chi.URLParam(r, "eventId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewGuestResponse(guest))
}

func (h *GuestHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	guests, err := h.service.ListByEvent(r.Context(), userID, chi.URLParam(r, "eventId"))
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

func (h *GuestHandler) GetByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	guest, err := h.service.GetByID(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "guestId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGuestResponse(guest))
}

func (h *GuestHandler) GetQRCode(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	guest, err := h.service.GetByID(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "guestId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGuestQRCodeResponse(guest))
}

func (h *GuestHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.UpdateGuestRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	guest, err := h.service.Update(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "guestId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGuestResponse(guest))
}

func (h *GuestHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	if err := h.service.Delete(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "guestId")); err != nil {
		apphttp.MapError(w, err)
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}
