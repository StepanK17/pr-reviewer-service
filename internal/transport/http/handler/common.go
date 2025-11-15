package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	domainErrors "github.com/StepanK17/pr-reviewer-service/internal/domain/errors"
	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/dto"
)

// respondJSON отправляет JSON ответ
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Логируем ошибку, но не можем изменить статус код
		http.Error(w, "internal server error", http.StatusInternalServerError)
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

// handleUseCaseError обрабатывает ошибки из usecase слоя
func handleUseCaseError(w http.ResponseWriter, err error) {
	var domainErr *domainErrors.DomainError
	if errors.As(err, &domainErr) {
		// Определяем HTTP статус код по коду ошибки
		status := getStatusCodeByErrorCode(domainErr.Code)
		respondError(w, status, domainErr.Code, domainErr.Message)
		return
	}

	// Неизвестная ошибка
	respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

// getStatusCodeByErrorCode возвращает HTTP статус код по коду доменной ошибки
func getStatusCodeByErrorCode(code string) int {
	switch code {
	case "TEAM_EXISTS", "PR_EXISTS":
		return http.StatusBadRequest
	case "PR_MERGED", "NOT_ASSIGNED", "NO_CANDIDATE":
		return http.StatusConflict
	case "NOT_FOUND":
		return http.StatusNotFound
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "INVALID_INPUT":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
