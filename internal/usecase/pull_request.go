package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
	domainErrors "github.com/StepanK17/pr-reviewer-service/internal/domain/errors"
	"github.com/StepanK17/pr-reviewer-service/internal/repository"
)

// PullRequestUseCase реализует бизнес-логику для PR
type PullRequestUseCase struct {
	prRepo    repository.PullRequestRepository
	userRepo  repository.UserRepository
	txManager repository.TransactionManager
}

// NewPullRequestUseCase создает новый usecase для PR
func NewPullRequestUseCase(
	prRepo repository.PullRequestRepository,
	userRepo repository.UserRepository,
	txManager repository.TransactionManager,
) *PullRequestUseCase {
	return &PullRequestUseCase{
		prRepo:    prRepo,
		userRepo:  userRepo,
		txManager: txManager,
	}
}

// CreatePullRequest создает PR и автоматически назначает ревьюверов
func (uc *PullRequestUseCase) CreatePullRequest(
	ctx context.Context,
	prID, prName, authorID string,
) (*entity.PullRequest, error) {
	var result *entity.PullRequest

	err := uc.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		// Проверяем существование PR
		exists, err := uc.prRepo.Exists(ctx, prID)
		if err != nil {
			return fmt.Errorf("failed to check PR existence: %w", err)
		}

		if exists {
			return domainErrors.NewDomainError(
				"PR_EXISTS",
				"PR id already exists",
				domainErrors.ErrPRExists,
			)
		}

		// Получаем автора
		author, err := uc.userRepo.GetByID(ctx, authorID)
		if err != nil {
			if errors.Is(err, domainErrors.ErrNotFound) {
				return domainErrors.NewDomainError(
					"NOT_FOUND",
					"author not found",
					domainErrors.ErrNotFound,
				)
			}
			return fmt.Errorf("failed to get author: %w", err)
		}

		// Получаем активных пользователей команды автора (исключая автора)
		reviewers, err := uc.selectReviewers(ctx, author.TeamName, authorID)
		if err != nil {
			return fmt.Errorf("failed to select reviewers: %w", err)
		}

		// Создаем PR
		pr := &entity.PullRequest{
			PullRequestID:     prID,
			PullRequestName:   prName,
			AuthorID:          authorID,
			Status:            entity.PRStatusOpen,
			AssignedReviewers: reviewers,
			CreatedAt:         time.Now(),
			MergedAt:          nil,
		}

		if err := uc.prRepo.Create(ctx, pr); err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}

		result = pr
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// MergePullRequest помечает PR как MERGED (идемпотентная операция)
func (uc *PullRequestUseCase) MergePullRequest(ctx context.Context, prID string) (*entity.PullRequest, error) {
	var result *entity.PullRequest

	err := uc.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		// Получаем PR
		pr, err := uc.prRepo.GetByID(ctx, prID)
		if err != nil {
			if errors.Is(err, domainErrors.ErrNotFound) {
				return domainErrors.NewDomainError(
					"NOT_FOUND",
					"PR not found",
					domainErrors.ErrNotFound,
				)
			}
			return fmt.Errorf("failed to get PR: %w", err)
		}

		// Если уже merged, возвращаем текущее состояние (идемпотентность)
		if pr.Status == entity.PRStatusMerged {
			result = pr
			return nil
		}

		// Помечаем как merged
		now := time.Now()
		pr.Status = entity.PRStatusMerged
		pr.MergedAt = &now

		if err := uc.prRepo.Update(ctx, pr); err != nil {
			return fmt.Errorf("failed to update PR: %w", err)
		}

		result = pr
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ReassignReviewer переназначает ревьювера
func (uc *PullRequestUseCase) ReassignReviewer(
	ctx context.Context,
	prID, oldUserID string,
) (*entity.PullRequest, string, error) {
	var result *entity.PullRequest
	var newReviewerID string

	err := uc.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		// Получаем PR
		pr, err := uc.prRepo.GetByID(ctx, prID)
		if err != nil {
			if errors.Is(err, domainErrors.ErrNotFound) {
				return domainErrors.NewDomainError(
					"NOT_FOUND",
					"PR not found",
					domainErrors.ErrNotFound,
				)
			}
			return fmt.Errorf("failed to get PR: %w", err)
		}

		// Проверяем, что PR не merged
		if pr.Status == entity.PRStatusMerged {
			return domainErrors.NewDomainError(
				"PR_MERGED",
				"cannot reassign on merged PR",
				domainErrors.ErrPRMerged,
			)
		}

		// Проверяем, что oldUserID назначен ревьювером
		oldUserIndex := -1
		for i, reviewerID := range pr.AssignedReviewers {
			if reviewerID == oldUserID {
				oldUserIndex = i
				break
			}
		}

		if oldUserIndex == -1 {
			return domainErrors.NewDomainError(
				"NOT_ASSIGNED",
				"reviewer is not assigned to this PR",
				domainErrors.ErrNotAssigned,
			)
		}

		// Получаем старого ревьювера для определения его команды
		oldReviewer, err := uc.userRepo.GetByID(ctx, oldUserID)
		if err != nil {
			if errors.Is(err, domainErrors.ErrNotFound) {
				return domainErrors.NewDomainError(
					"NOT_FOUND",
					"old reviewer not found",
					domainErrors.ErrNotFound,
				)
			}
			return fmt.Errorf("failed to get old reviewer: %w", err)
		}

		// Получаем активных пользователей команды заменяемого ревьювера
		candidates, err := uc.userRepo.GetActiveByTeam(ctx, oldReviewer.TeamName)
		if err != nil {
			return fmt.Errorf("failed to get team members: %w", err)
		}

		// Фильтруем кандидатов (исключаем автора и уже назначенных ревьюверов)
		var availableCandidates []*entity.User
		for _, candidate := range candidates {
			// Исключаем автора
			if candidate.UserID == pr.AuthorID {
				continue
			}

			// Исключаем уже назначенных ревьюверов
			alreadyAssigned := false
			for _, reviewerID := range pr.AssignedReviewers {
				if candidate.UserID == reviewerID {
					alreadyAssigned = true
					break
				}
			}

			if !alreadyAssigned {
				availableCandidates = append(availableCandidates, candidate)
			}
		}

		// Проверяем наличие доступных кандидатов
		if len(availableCandidates) == 0 {
			return domainErrors.NewDomainError(
				"NO_CANDIDATE",
				"no active replacement candidate in team",
				domainErrors.ErrNoCandidate,
			)
		}

		// Выбираем случайного кандидата
		newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]
		newReviewerID = newReviewer.UserID

		// Заменяем ревьювера
		pr.AssignedReviewers[oldUserIndex] = newReviewerID

		// Обновляем PR
		if err := uc.prRepo.Update(ctx, pr); err != nil {
			return fmt.Errorf("failed to update PR: %w", err)
		}

		result = pr
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	return result, newReviewerID, nil
}

// selectReviewers выбирает до 2 активных ревьюверов из команды (исключая автора)
func (uc *PullRequestUseCase) selectReviewers(ctx context.Context, teamName, authorID string) ([]string, error) {
	// Получаем активных пользователей команды
	users, err := uc.userRepo.GetActiveByTeam(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get active team members: %w", err)
	}

	// Фильтруем автора
	var candidates []*entity.User
	for _, user := range users {
		if user.UserID != authorID {
			candidates = append(candidates, user)
		}
	}

	// Если кандидатов нет, возвращаем пустой список
	if len(candidates) == 0 {
		return []string{}, nil
	}

	// Перемешиваем кандидатов для случайного выбора
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Выбираем до 2 ревьюверов
	maxReviewers := 2
	if len(candidates) < maxReviewers {
		maxReviewers = len(candidates)
	}

	reviewers := make([]string, maxReviewers)
	for i := 0; i < maxReviewers; i++ {
		reviewers[i] = candidates[i].UserID
	}

	return reviewers, nil
}
