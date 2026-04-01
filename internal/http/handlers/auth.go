package handlers

import (
	nethttp "net/http"

	"myevent-back/internal/dto"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/http/middleware"
	"myevent-back/internal/models"
	"myevent-back/internal/services"
)

type AuthHandler struct {
	service        *services.AuthService
	accountService *services.AccountService
}

func NewAuthHandler(service *services.AuthService, accountService *services.AccountService) *AuthHandler {
	return &AuthHandler{
		service:        service,
		accountService: accountService,
	}
}

func (h *AuthHandler) Register(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.RegisterRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	user, token, err := h.service.Register(
		r.Context(),
		request.Name,
		request.Email,
		request.Password,
		request.ContactPhone,
		request.AcceptedTerms,
		request.MarketingOptIn,
		models.UserAttribution{
			UTMSource:   request.UTMSource,
			UTMMedium:   request.UTMMedium,
			UTMCampaign: request.UTMCampaign,
			UTMTerm:     request.UTMTerm,
			UTMContent:  request.UTMContent,
		},
	)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusCreated, dto.NewAuthResponse(user, token, "Conta criada com sucesso."))
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

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewAuthResponse(user, token, "Login realizado com sucesso."))
}

func (h *AuthHandler) ForgotPassword(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.ForgotPasswordRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	message, err := h.service.ForgotPassword(r.Context(), request.Email)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.MessageResponse{Message: message})
}

func (h *AuthHandler) ResetPassword(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request dto.ResetPasswordRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	message, err := h.service.ResetPassword(r.Context(), request.Token, request.Password)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.MessageResponse{Message: message})
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
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

	user, err := h.service.Me(r.Context(), userID)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewUserResponse(user))
}

func (h *AuthHandler) UpdateMe(w nethttp.ResponseWriter, r *nethttp.Request) {
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

	var request dto.UpdateProfileRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, request.Name, request.ContactPhone)
	if err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.NewProfileResponse(user, "Dados atualizados com sucesso."))
}

func (h *AuthHandler) DeleteMe(w nethttp.ResponseWriter, r *nethttp.Request) {
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

	var request dto.DeleteAccountRequest
	if !apphttp.DecodeJSON(w, r, &request) {
		return
	}

	if err := h.accountService.Delete(r.Context(), userID, request.Email); err != nil {
		apphttp.MapError(w, err)
		return
	}

	apphttp.WriteJSON(w, nethttp.StatusOK, dto.MessageResponse{Message: "Conta excluida com sucesso."})
}
