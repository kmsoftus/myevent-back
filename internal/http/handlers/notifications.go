package handlers

import (
	nethttp "net/http"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type NotificationHandler struct {
	service *services.OrganizerNotificationService
}

func NewNotificationHandler(service *services.OrganizerNotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) RegisterDeviceToken(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteErrorResponse(
			w,
			nethttp.StatusUnauthorized,
			"Sessao invalida. Faca login novamente.",
			"auth_session_invalid",
			nil,
		)
		return
	}

	var request dto.RegisterDeviceTokenRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	if err := h.service.RegisterDeviceToken(r.Context(), userID, request.Token, request.Platform); err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(
		w,
		nethttp.StatusCreated,
		dto.MessageResponse{Message: "Token de notificacao registrado com sucesso."},
	)
}
