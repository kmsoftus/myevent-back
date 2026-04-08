package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type GiftTransactionHandler struct {
	service *services.GiftTransactionService
}

func NewGiftTransactionHandler(service *services.GiftTransactionService) *GiftTransactionHandler {
	return &GiftTransactionHandler{service: service}
}

func (h *GiftTransactionHandler) ReservePublic(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.CreateGiftTransactionRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	details, err := h.service.ReserveBySlug(r.Context(), chi.URLParam(r, "slug"), chi.URLParam(r, "giftId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewGiftTransactionResponse(details.Transaction, details.Gift))
}

func (h *GiftTransactionHandler) PixPublic(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.CreateGiftTransactionRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	details, err := h.service.RegisterPixBySlug(r.Context(), chi.URLParam(r, "slug"), chi.URLParam(r, "giftId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewGiftTransactionResponse(details.Transaction, details.Gift))
}

func (h *GiftTransactionHandler) ListByEvent(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	page, pageSize := apphttp.ReadPagination(r)
	details, err := h.service.ListByEvent(r.Context(), userID, chi.URLParam(r, "eventId"), page, pageSize)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GiftTransactionResponse, 0, len(details.Items))
	for _, item := range details.Items {
		response = append(response, dto.NewGiftTransactionResponse(item.Transaction, item.Gift))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.PaginatedResponse[dto.GiftTransactionResponse]{
		Items:      response,
		Total:      details.Total,
		Page:       details.Page,
		PageSize:   details.PageSize,
		TotalPages: details.TotalPages,
	})
}

func (h *GiftTransactionHandler) Confirm(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.UpdateGiftTransactionStatusRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	details, err := h.service.Confirm(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "transactionId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGiftTransactionResponse(details.Transaction, details.Gift))
}

func (h *GiftTransactionHandler) Cancel(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.UpdateGiftTransactionStatusRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	details, err := h.service.Cancel(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "transactionId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewGiftTransactionResponse(details.Transaction, details.Gift))
}
