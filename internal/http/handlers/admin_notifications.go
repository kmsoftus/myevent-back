package handlers

import (
	"fmt"
	nethttp "net/http"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/services"
)

type AdminNotificationHandler struct {
	service *services.OrganizerNotificationService
}

func NewAdminNotificationHandler(service *services.OrganizerNotificationService) *AdminNotificationHandler {
	return &AdminNotificationHandler{service: service}
}

func (h *AdminNotificationHandler) SendPromotional(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req dto.SendPromotionalNotificationRequest
	if !apphttp.DecodeJSON(w, r, &req) {
		return
	}

	sent, errs := h.service.SendPromotionalNotification(r.Context(), req.Title, req.Body)

	if len(errs) > 0 && sent == 0 {
		apphttp.MapError(w, errs[0])
		return
	}

	msg := fmt.Sprintf("Notificacao enviada para %d dispositivo(s).", sent)
	if len(errs) > 0 {
		msg = fmt.Sprintf("Notificacao enviada para %d dispositivo(s), %d falha(s).", sent, len(errs))
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.SendPromotionalNotificationResponse{
		Message:  msg,
		Sent:     sent,
		Failures: len(errs),
	})
}
