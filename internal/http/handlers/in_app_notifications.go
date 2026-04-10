package handlers

import (
	nethttp "net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type InAppNotificationHandler struct {
	service *services.OrganizerNotificationService
}

func NewInAppNotificationHandler(service *services.OrganizerNotificationService) *InAppNotificationHandler {
	return &InAppNotificationHandler{service: service}
}

func (h *InAppNotificationHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteErrorResponse(w, nethttp.StatusUnauthorized, "Sessao invalida. Faca login novamente.", "auth_session_invalid", nil)
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	list, unread, err := h.service.ListNotifications(r.Context(), userID, limit, offset)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	resp := dto.ListNotificationsResponse{
		Notifications: make([]dto.NotificationResponse, 0, len(list)),
		Unread:        unread,
	}
	for _, n := range list {
		resp.Notifications = append(resp.Notifications, dto.NotificationFromModel(n))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, resp)
}

func (h *InAppNotificationHandler) MarkRead(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteErrorResponse(w, nethttp.StatusUnauthorized, "Sessao invalida. Faca login novamente.", "auth_session_invalid", nil)
		return
	}

	id := chi.URLParam(r, "notificationId")
	if id == "" {
		apphttp.WriteErrorResponse(w, nethttp.StatusBadRequest, "ID da notificacao nao informado.", "notification_id_required", nil)
		return
	}

	if err := h.service.MarkNotificationRead(r.Context(), id, userID); err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.MessageResponse{Message: "Notificacao marcada como lida."})
}

func (h *InAppNotificationHandler) MarkAllRead(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteErrorResponse(w, nethttp.StatusUnauthorized, "Sessao invalida. Faca login novamente.", "auth_session_invalid", nil)
		return
	}

	if err := h.service.MarkAllNotificationsRead(r.Context(), userID); err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.MessageResponse{Message: "Todas as notificacoes marcadas como lidas."})
}
