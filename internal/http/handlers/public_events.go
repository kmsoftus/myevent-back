package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/services"
)

type PublicEventHandler struct {
	service *services.EventService
}

func NewPublicEventHandler(service *services.EventService) *PublicEventHandler {
	return &PublicEventHandler{service: service}
}

func (h *PublicEventHandler) GetBySlug(w nethttp.ResponseWriter, r *nethttp.Request) {
	event, err := h.service.GetPublishedBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewPublicEventResponse(event))
}
