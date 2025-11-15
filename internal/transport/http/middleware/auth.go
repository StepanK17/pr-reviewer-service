package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/dto"
)

// AdminAuth проверяет админский токен
func AdminAuth(adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization header")
				return
			}

			token := strings.TrimPrefix(authHeader, prefix)

			if token != adminToken {
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid admin token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// respondError отправляет ошибку в формате API
func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(response)
}
