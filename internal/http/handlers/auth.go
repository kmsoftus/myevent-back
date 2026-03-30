package handlers

import (
	nethttp "net/http"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/services"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.RegisterRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	user, token, err := h.service.Register(r.Context(), request.Name, request.Email, request.Password)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewAuthResponse(user, token))
}

func (h *AuthHandler) Login(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.LoginRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	user, token, err := h.service.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewAuthResponse(user, token))
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apphttp.WriteError(w, nethttp.StatusUnauthorized, "missing authenticated user")
		return
	}

	user, err := h.service.Me(r.Context(), userID)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewUserResponse(user))
}
