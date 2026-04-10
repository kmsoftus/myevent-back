package dto

import (
	"time"

	"myevent-back/internal/models"
)

// ---------- in-app notifications ----------

type NotificationResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	ReadAt    *time.Time        `json:"read_at"`
	CreatedAt time.Time         `json:"created_at"`
}

func NotificationFromModel(n *models.Notification) NotificationResponse {
	data := n.Data
	if data == nil {
		data = map[string]string{}
	}
	return NotificationResponse{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Data:      data,
		ReadAt:    n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
}

type ListNotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Unread        int                    `json:"unread"`
}

// ---------- push device tokens ----------

type RegisterDeviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

type SendPromotionalNotificationRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type SendPromotionalNotificationResponse struct {
	Message   string `json:"message"`
	Sent      int    `json:"sent"`
	Failures  int    `json:"failures"`
}
