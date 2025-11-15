package handler

import (
	"net/http"

	"github.com/StepanK17/pr-reviewer-service/internal/usecase"
)

// StatisticsHandler обрабатывает запросы для статистики
type StatisticsHandler struct {
	statsUseCase *usecase.StatisticsUseCase
}

// NewStatisticsHandler создает новый handler для статистики
func NewStatisticsHandler(statsUseCase *usecase.StatisticsUseCase) *StatisticsHandler {
	return &StatisticsHandler{
		statsUseCase: statsUseCase,
	}
}

// GetStatistics обрабатывает GET /statistics
func (h *StatisticsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUseCase.GetStatistics(r.Context())
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
