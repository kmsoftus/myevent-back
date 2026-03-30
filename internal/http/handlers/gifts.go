package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type GiftHandler struct {
	service *services.GiftService
}

func NewGiftHandler(service *services.GiftService) *GiftHandler {
	return &GiftHandler{service: service}
}

func (h *GiftHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.CreateGiftRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	gift, err := h.service.Create(r.Context(), userID, chi.URLParam(r, "eventId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewGiftResponse(gift))
}

func (h *GiftHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	gifts, err := h.service.ListByEvent(r.Context(), userID, chi.URLParam(r, "eventId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GiftResponse, 0, len(gifts))
	for _, gift := range gifts {
		response = append(response, dto.NewGiftResponse(gift))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}

func (h *GiftHandler) ListPublic(w nethttp.ResponseWriter, r *nethttp.Request) {
	gifts, err := h.service.ListPublicBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GiftResponse, 0, len(gifts))
	for _, gift := range gifts {
		response = append(response, dto.NewGiftResponse(gift))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}

func (h *GiftHandler) GetByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	gift, err := h.service.GetByID(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "giftId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGiftResponse(gift))
}

func (h *GiftHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.UpdateGiftRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	gift, err := h.service.Update(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "giftId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGiftResponse(gift))
}

func (h *GiftHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	if err := h.service.Delete(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "giftId")); err != nil {
		apphttp.MapError(w, err)
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}
