package middleware

import (
	"context"
	"strings"

	nethttp "net/http"

	"myevent-back/internal/auth"
	apphttp "myevent-back/internal/http"
)

type contextKey string

const userIDContextKey contextKey = "userID"

func Authenticator(jwtManager *auth.JWTManager) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				apphttp.WriteErrorResponse(
					w,
					nethttp.StatusUnauthorized,
					"Token de acesso nao informado.",
					"auth_token_missing",
					nil,
				)
				return
			}

			claims, err := jwtManager.ParseToken(token)
			if err != nil {
				apphttp.WriteErrorResponse(
					w,
					nethttp.StatusUnauthorized,
					"Token de acesso invalido ou expirado.",
					"auth_token_invalid",
					nil,
				)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok && userID != ""
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
