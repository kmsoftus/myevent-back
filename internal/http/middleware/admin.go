package middleware

import (
	"context"
	"strings"

	nethttp "net/http"

	"myevent-back/internal/auth"
	apphttp "myevent-back/internal/http"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type adminUserRepo interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
}

// AdminAuthenticator validates the JWT token and checks that the user's email
// is in the allowlist. It sets the userID in context like Authenticator does.
func AdminAuthenticator(jwtManager *auth.JWTManager, userRepo adminUserRepo, allowedEmails []string) func(nethttp.Handler) nethttp.Handler {
	allowed := make(map[string]struct{}, len(allowedEmails))
	for _, e := range allowedEmails {
		allowed[strings.ToLower(strings.TrimSpace(e))] = struct{}{}
	}

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				apphttp.WriteErrorResponse(w, nethttp.StatusUnauthorized, "Token de acesso nao informado.", "auth_token_missing", nil)
				return
			}

			claims, err := jwtManager.ParseToken(token)
			if err != nil {
				apphttp.WriteErrorResponse(w, nethttp.StatusUnauthorized, "Token de acesso invalido ou expirado.", "auth_token_invalid", nil)
				return
			}

			if len(allowed) > 0 {
				user, err := userRepo.GetByID(r.Context(), claims.UserID)
				if err != nil {
					if err == repositories.ErrNotFound {
						apphttp.WriteErrorResponse(w, nethttp.StatusForbidden, "Acesso negado.", "admin_forbidden", nil)
						return
					}
					apphttp.WriteErrorResponse(w, nethttp.StatusInternalServerError, "Erro interno.", "internal_error", nil)
					return
				}

				if _, ok := allowed[strings.ToLower(strings.TrimSpace(user.Email))]; !ok {
					apphttp.WriteErrorResponse(w, nethttp.StatusForbidden, "Acesso negado.", "admin_forbidden", nil)
					return
				}
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
