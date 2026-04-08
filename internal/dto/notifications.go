package dto

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
