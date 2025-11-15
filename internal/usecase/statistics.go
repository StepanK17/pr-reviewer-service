package usecase

import (
	"context"
	"fmt"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
	"github.com/StepanK17/pr-reviewer-service/internal/repository"
)

// StatisticsUseCase реализует бизнес-логику для статистики
type StatisticsUseCase struct {
	statsRepo repository.StatisticsRepository
}

// NewStatisticsUseCase создает новый usecase для статистики
func NewStatisticsUseCase(statsRepo repository.StatisticsRepository) *StatisticsUseCase {
	return &StatisticsUseCase{
		statsRepo: statsRepo,
	}
}

// GetStatistics возвращает общую статистику системы
func (uc *StatisticsUseCase) GetStatistics(ctx context.Context) (*entity.Statistics, error) {
	stats, err := uc.statsRepo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}
