package dto

import "myevent-back/internal/models"

type RegisterRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	ContactPhone   string `json:"contact_phone"`
	AcceptedTerms  bool   `json:"accepted_terms"`
	MarketingOptIn bool   `json:"marketing_opt_in"`
	Password       string `json:"password"`
	UTMSource      string `json:"utm_source"`
	UTMMedium      string `json:"utm_medium"`
	UTMCampaign    string `json:"utm_campaign"`
	UTMTerm        string `json:"utm_term"`
	UTMContent     string `json:"utm_content"`
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

type DeleteAccountRequest struct {
	Email string `json:"email"`
}

type UpdateProfileRequest struct {
	Name            string `json:"name"`
	ContactPhone    string `json:"contact_phone"`
	ProfilePhotoURL string `json:"profile_photo_url"`
}

type UserResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	ContactPhone    string `json:"contact_phone,omitempty"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
}

type AuthResponse struct {
	Message string       `json:"message,omitempty"`
	User    UserResponse `json:"user"`
	Token   string       `json:"token"`
}

type ProfileResponse struct {
	Message string       `json:"message,omitempty"`
	User    UserResponse `json:"user"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

func NewUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		ContactPhone:    user.ContactPhone,
		ProfilePhotoURL: user.ProfilePhotoURL,
	}
}

func NewAuthResponse(user *models.User, token, message string) AuthResponse {
	return AuthResponse{
		Message: message,
		User:    NewUserResponse(user),
		Token:   token,
	}
}

func NewProfileResponse(user *models.User, message string) ProfileResponse {
	return ProfileResponse{
		Message: message,
		User:    NewUserResponse(user),
	}
}
