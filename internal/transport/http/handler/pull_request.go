package handler

import (
	"encoding/json"
	"net/http"

	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/dto"
	"github.com/StepanK17/pr-reviewer-service/internal/usecase"
)

// PullRequestHandler обрабатывает запросы для PR
type PullRequestHandler struct {
	prUseCase *usecase.PullRequestUseCase
}

// NewPullRequestHandler создает новый handler для PR
func NewPullRequestHandler(prUseCase *usecase.PullRequestUseCase) *PullRequestHandler {
	return &PullRequestHandler{
		prUseCase: prUseCase,
	}
}

// CreatePR обрабатывает POST /pullRequest/create
func (h *PullRequestHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	// Валидация
	if req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "pull_request_id, pull_request_name and author_id are required")
		return
	}

	// Создаем PR
	pr, err := h.prUseCase.CreatePullRequest(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	// Формируем ответ
	response := dto.CreatePRResponse{
		PR: dto.ToPullRequestDTO(pr),
	}

	respondJSON(w, http.StatusCreated, response)
}

// MergePR обрабатывает POST /pullRequest/merge
func (h *PullRequestHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req dto.MergePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	// Валидация
	if req.PullRequestID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "pull_request_id is required")
		return
	}

	pr, err := h.prUseCase.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response := dto.MergePRResponse{
		PR: dto.ToPullRequestDTO(pr),
	}

	respondJSON(w, http.StatusOK, response)
}

// ReassignReviewer обрабатывает POST /pullRequest/reassign
func (h *PullRequestHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req dto.ReassignRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.PullRequestID == "" || req.OldUserID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "pull_request_id and old_user_id are required")
		return
	}

	pr, newReviewerID, err := h.prUseCase.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response := dto.ReassignResponse{
		PR:         dto.ToPullRequestDTO(pr),
		ReplacedBy: newReviewerID,
	}

	respondJSON(w, http.StatusOK, response)
}
