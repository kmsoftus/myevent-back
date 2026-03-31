package dto

import "myevent-back/internal/models"

type RegisterRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMTerm     string `json:"utm_term"`
	UTMContent  string `json:"utm_content"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthResponse struct {
	Message string       `json:"message,omitempty"`
	User    UserResponse `json:"user"`
	Token   string       `json:"token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

func NewUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}
}

func NewAuthResponse(user *models.User, token, message string) AuthResponse {
	return AuthResponse{
		Message: message,
		User:    NewUserResponse(user),
		Token:   token,
	}
}
