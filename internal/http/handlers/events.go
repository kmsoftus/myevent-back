package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type EventHandler struct {
	service *services.EventService
}

func NewEventHandler(service *services.EventService) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.CreateEventRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	event, err := h.service.Create(r.Context(), userID, request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewEventResponse(event))
}

func (h *EventHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	query := r.URL.Query()
	input := dto.ListEventsRequest{
		Query:  query.Get("q"),
		Status: query.Get("status"),
		Sort:   query.Get("sort"),
	}

	events, err := h.service.ListByUser(r.Context(), userID, input)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.EventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, dto.NewEventResponse(event))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}

func (h *EventHandler) GetByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	event, err := h.service.GetByIDForUser(r.Context(), userID, chi.URLParam(r, "eventId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewEventResponse(event))
}

func (h *EventHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.UpdateEventRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	event, err := h.service.Update(r.Context(), userID, chi.URLParam(r, "eventId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewEventResponse(event))
}

func (h *EventHandler) UpdateStatus(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.UpdateEventStatusRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	event, err := h.service.UpdateStatus(r.Context(), userID, chi.URLParam(r, "eventId"), request.Status)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewEventResponse(event))
}

func (h *EventHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	if err := h.service.Delete(r.Context(), userID, chi.URLParam(r, "eventId")); err != nil {
		apphttp.MapError(w, err)
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}
