package handlers

import (
	"errors"
	"mime/multipart"
	nethttp "net/http"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type UploadHandler struct {
	service *services.UploadService
}

func NewUploadHandler(service *services.UploadService) *UploadHandler {
	return &UploadHandler{service: service}
}

func (h *UploadHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	r.Body = nethttp.MaxBytesReader(w, r.Body, h.service.MaxBodySizeBytes())
	if err := r.ParseMultipartForm(h.service.MaxBodySizeBytes()); err != nil {
		if errors.Is(err, multipart.ErrMessageTooLarge) {
			apphttp.WriteError(w, nethttp.StatusRequestEntityTooLarge, "multipart body is too large")
			return
		}

		apphttp.WriteError(w, nethttp.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apphttp.WriteError(w, nethttp.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	upload, err := h.service.Upload(r.Context(), r.FormValue("folder"), header.Filename, file)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, upload)
}

func (h *UploadHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	var request dto.DeleteUploadRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	if err := h.service.Delete(r.Context(), request.Key); err != nil {
		apphttp.MapError(w, err)
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}
