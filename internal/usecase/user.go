package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
	domainErrors "github.com/StepanK17/pr-reviewer-service/internal/domain/errors"
	"github.com/StepanK17/pr-reviewer-service/internal/repository"
)

// UserUseCase реализует бизнес-логику для пользователей
type UserUseCase struct {
	userRepo repository.UserRepository
	prRepo   repository.PullRequestRepository
}

// NewUserUseCase создает новый usecase для пользователей
func NewUserUseCase(
	userRepo repository.UserRepository,
	prRepo repository.PullRequestRepository,
) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

// SetIsActive устанавливает флаг активности пользователя
func (uc *UserUseCase) SetIsActive(ctx context.Context, userID string, isActive bool) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNotFound) {
			return nil, domainErrors.NewDomainError(
				"NOT_FOUND",
				"user not found",
				domainErrors.ErrNotFound,
			)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.IsActive = isActive
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// GetUserReviews возвращает список Pов, где пользователь назначен ревьювером
func (uc *UserUseCase) GetUserReviews(ctx context.Context, userID string) ([]*entity.PullRequestShort, error) {
	// Проверяем существование пользователя
	_, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNotFound) {
			return nil, domainErrors.NewDomainError(
				"NOT_FOUND",
				"user not found",
				domainErrors.ErrNotFound,
			)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Получаем PRы
	prs, err := uc.prRepo.GetByReviewer(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reviews: %w", err)
	}

	// Если пользователь не назначен ни на один PR, возвращаем пустой список
	if prs == nil {
		prs = []*entity.PullRequestShort{}
	}

	return prs, nil
}
