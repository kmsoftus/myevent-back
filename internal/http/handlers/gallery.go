package handlers

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type GalleryHandler struct {
	service *services.GalleryService
}

func NewGalleryHandler(service *services.GalleryService) *GalleryHandler {
	return &GalleryHandler{service: service}
}

func (h *GalleryHandler) Add(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.AddGalleryPhotoRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	photo, err := h.service.AddPhoto(r.Context(), userID, chi.URLParam(r, "eventId"), request)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewGalleryPhotoResponse(photo))
}

func (h *GalleryHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	photos, err := h.service.ListByEvent(r.Context(), userID, chi.URLParam(r, "eventId"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GalleryPhotoResponse, 0, len(photos))
	for _, p := range photos {
		response = append(response, dto.NewGalleryPhotoResponse(p))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}

func (h *GalleryHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	if err := h.service.DeletePhoto(r.Context(), userID, chi.URLParam(r, "eventId"), chi.URLParam(r, "photoId")); err != nil {
		apphttp.MapError(w, err)
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

// ListPublic is called without authentication (public route).
type PublicGalleryHandler struct {
	service *services.GalleryService
}

func NewPublicGalleryHandler(service *services.GalleryService) *PublicGalleryHandler {
	return &PublicGalleryHandler{service: service}
}

func (h *PublicGalleryHandler) ListBySlug(w nethttp.ResponseWriter, r *nethttp.Request) {
	photos, err := h.service.ListPublicBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	response := make([]dto.GalleryPhotoResponse, 0, len(photos))
	for _, p := range photos {
		response = append(response, dto.NewGalleryPhotoResponse(p))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, response)
}
