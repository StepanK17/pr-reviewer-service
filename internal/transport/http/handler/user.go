package handler

import (
	"encoding/json"
	"net/http"

	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/dto"
	"github.com/StepanK17/pr-reviewer-service/internal/usecase"
)

// UserHandler обрабатывает запросы для пользователей
type UserHandler struct {
	userUseCase *usecase.UserUseCase
}

// NewUserHandler создает новый handler для пользователей
func NewUserHandler(userUseCase *usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

// SetIsActive обрабатывает POST /users/setIsActive
func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req dto.SetIsActiveRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.UserID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "user_id is required")
		return
	}

	user, err := h.userUseCase.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response := dto.SetIsActiveResponse{
		User: dto.ToUserDTO(user),
	}

	respondJSON(w, http.StatusOK, response)
}

// GetReview обрабатывает GET /users/getReview
func (h *UserHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "user_id query parameter is required")
		return
	}

	prs, err := h.userUseCase.GetUserReviews(r.Context(), userID)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response := dto.GetUserReviewsResponse{
		UserID:       userID,
		PullRequests: dto.ToPullRequestShortDTOs(prs),
	}

	respondJSON(w, http.StatusOK, response)
}
