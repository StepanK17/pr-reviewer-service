package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
	domainErrors "github.com/StepanK17/pr-reviewer-service/internal/domain/errors"
)

// DeactivateTeamMembersResult результат массовой деактивации
type DeactivateTeamMembersResult struct {
	DeactivatedCount int
	ReassignedPRs    int
	UserIDs          []string
}

// DeactivateTeamMembers массово деактивирует пользователей команды и переназначает их PR
func (uc *TeamUseCase) DeactivateTeamMembers(ctx context.Context, teamName string) (*DeactivateTeamMembersResult, error) {
	var result DeactivateTeamMembersResult

	err := uc.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		// 1. Проверяем существование команды
		_, err := uc.teamRepo.GetByName(ctx, teamName)
		if err != nil {
			if errors.Is(err, domainErrors.ErrNotFound) {
				return domainErrors.NewDomainError(
					"NOT_FOUND",
					"team not found",
					domainErrors.ErrNotFound,
				)
			}
			return fmt.Errorf("failed to get team: %w", err)
		}

		// 2. Получаем всех пользователей команды
		users, err := uc.userRepo.GetByTeam(ctx, teamName)
		if err != nil {
			return fmt.Errorf("failed to get team members: %w", err)
		}

		// 3. Деактивируем активных пользователей
		deactivatedUserIDs := make([]string, 0)
		now := time.Now()

		for _, user := range users {
			if user.IsActive {
				user.IsActive = false
				user.UpdatedAt = now
				if err := uc.userRepo.Update(ctx, user); err != nil {
					return fmt.Errorf("failed to deactivate user %s: %w", user.UserID, err)
				}
				deactivatedUserIDs = append(deactivatedUserIDs, user.UserID)
			}
		}

		result.DeactivatedCount = len(deactivatedUserIDs)
		result.UserIDs = deactivatedUserIDs

		// 4. Находим и переназначаем открытые PR
		reassignedCount := 0
		for _, userID := range deactivatedUserIDs {
			// Получаем PR где пользователь - ревьювер
			prs, err := uc.prRepo.GetByReviewer(ctx, userID)
			if err != nil {
				return fmt.Errorf("failed to get PRs for user %s: %w", userID, err)
			}

			// Переназначаем только открытые PR
			for _, prShort := range prs {
				if prShort.Status == entity.PRStatusOpen {
					// Получаем полный PR
					pr, err := uc.prRepo.GetByID(ctx, prShort.PullRequestID)
					if err != nil {
						return fmt.Errorf("failed to get PR %s: %w", prShort.PullRequestID, err)
					}

					// Пытаемся переназначить ревьювера
					if err := uc.reassignDeactivatedReviewer(ctx, pr, userID); err != nil {

						continue
					}
					reassignedCount++
				}
			}
		}

		result.ReassignedPRs = reassignedCount
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// reassignDeactivatedReviewer переназначает деактивированного ревьювера
func (uc *TeamUseCase) reassignDeactivatedReviewer(ctx context.Context, pr *entity.PullRequest, deactivatedUserID string) error {
	// Находим индекс деактивированного пользователя
	oldUserIndex := -1
	for i, reviewerID := range pr.AssignedReviewers {
		if reviewerID == deactivatedUserID {
			oldUserIndex = i
			break
		}
	}

	if oldUserIndex == -1 {
		return nil // Пользователь не назначен на этот PR
	}

	// Получаем команду деактивированного пользователя
	oldReviewer, err := uc.userRepo.GetByID(ctx, deactivatedUserID)
	if err != nil {
		return fmt.Errorf("failed to get old reviewer: %w", err)
	}

	// Получаем активных пользователей команды
	candidates, err := uc.userRepo.GetActiveByTeam(ctx, oldReviewer.TeamName)
	if err != nil {
		return fmt.Errorf("failed to get team members: %w", err)
	}

	// Фильтруем кандидатов
	var availableCandidates []*entity.User
	for _, candidate := range candidates {
		if candidate.UserID == pr.AuthorID {
			continue
		}

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

	// Если нет кандидатов, просто убираем деактивированного
	if len(availableCandidates) == 0 {
		// Удаляем ревьювера из списка
		pr.AssignedReviewers = append(pr.AssignedReviewers[:oldUserIndex], pr.AssignedReviewers[oldUserIndex+1:]...)
	} else {
		// Выбираем случайного кандидата
		newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]
		pr.AssignedReviewers[oldUserIndex] = newReviewer.UserID
	}

	// Обновляем PR
	if err := uc.prRepo.Update(ctx, pr); err != nil {
		return fmt.Errorf("failed to update PR: %w", err)
	}

	return nil
}
