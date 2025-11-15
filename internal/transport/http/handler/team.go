package handler

import (
	"encoding/json"
	"net/http"

	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/dto"
	"github.com/StepanK17/pr-reviewer-service/internal/usecase"
)

// TeamHandler обрабатывает запросы для команд
type TeamHandler struct {
	teamUseCase *usecase.TeamUseCase
}

// NewTeamHandler создает новый handler для команд
func NewTeamHandler(teamUseCase *usecase.TeamUseCase) *TeamHandler {
	return &TeamHandler{
		teamUseCase: teamUseCase,
	}
}

// CreateTeam обрабатывает POST /team/add
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.TeamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "team_name is required")
		return
	}

	if len(req.Members) == 0 {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "members is required")
		return
	}

	teamEntity := dto.ToTeamEntity(&req)
	team, err := h.teamUseCase.CreateTeam(r.Context(), teamEntity)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response := dto.CreateTeamResponse{
		Team: dto.ToTeamDTO(team),
	}

	respondJSON(w, http.StatusCreated, response)
}

// GetTeam обрабатывает GET /team/get
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "team_name query parameter is required")
		return
	}

	// Получаем команду
	team, err := h.teamUseCase.GetTeamWithMembers(r.Context(), teamName)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	// Формируем ответ
	response := dto.ToTeamDTO(team)
	respondJSON(w, http.StatusOK, response)
}

// DeactivateTeamMembers обрабатывает POST /team/deactivateMembers
func (h *TeamHandler) DeactivateTeamMembers(w http.ResponseWriter, r *http.Request) {
	var req dto.DeactivateTeamMembersRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.TeamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "team_name is required")
		return
	}

	result, err := h.teamUseCase.DeactivateTeamMembers(r.Context(), req.TeamName)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response := dto.DeactivateTeamMembersResponse{
		DeactivatedCount: result.DeactivatedCount,
		ReassignedPRs:    result.ReassignedPRs,
		UserIDs:          result.UserIDs,
	}

	respondJSON(w, http.StatusOK, response)
}
