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

// TeamUseCase реализует бизнес-логику для команд
type TeamUseCase struct {
	teamRepo  repository.TeamRepository
	userRepo  repository.UserRepository
	txManager repository.TransactionManager
	prRepo    repository.PullRequestRepository
}

// NewTeamUseCase создает новый usecase для команд
func NewTeamUseCase(
	teamRepo repository.TeamRepository,
	userRepo repository.UserRepository,
	txManager repository.TransactionManager,
	prRepo repository.PullRequestRepository,
) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:  teamRepo,
		userRepo:  userRepo,
		txManager: txManager,
		prRepo:    prRepo,
	}
}

// CreateTeam создает команду с участниками
func (uc *TeamUseCase) CreateTeam(ctx context.Context, teamWithMembers *entity.TeamWithMembers) (*entity.TeamWithMembers, error) {
	var result *entity.TeamWithMembers

	err := uc.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		// Проверяем, существует ли команда
		exists, err := uc.teamRepo.Exists(ctx, teamWithMembers.TeamName)
		if err != nil {
			return fmt.Errorf("failed to check team existence: %w", err)
		}

		if exists {
			return domainErrors.NewDomainError(
				"TEAM_EXISTS",
				"team_name already exists",
				domainErrors.ErrTeamExists,
			)
		}

		// Создаем команду
		team := &entity.Team{
			TeamName:  teamWithMembers.TeamName,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := uc.teamRepo.Create(ctx, team); err != nil {
			return fmt.Errorf("failed to create team: %w", err)
		}

		// Создаем/обновляем пользователей
		now := time.Now()
		users := make([]*entity.User, 0, len(teamWithMembers.Members))
		for _, member := range teamWithMembers.Members {
			user := &entity.User{
				UserID:    member.UserID,
				Username:  member.Username,
				TeamName:  teamWithMembers.TeamName,
				IsActive:  member.IsActive,
				CreatedAt: now,
				UpdatedAt: now,
			}
			users = append(users, user)
		}

		if err := uc.userRepo.UpsertBatch(ctx, users); err != nil {
			return fmt.Errorf("failed to upsert users: %w", err)
		}

		result = teamWithMembers
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetTeamWithMembers возвращает команду со списком участников
func (uc *TeamUseCase) GetTeamWithMembers(ctx context.Context, teamName string) (*entity.TeamWithMembers, error) {
	// Проверяем существование команды
	_, err := uc.teamRepo.GetByName(ctx, teamName)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNotFound) {
			return nil, domainErrors.NewDomainError(
				"NOT_FOUND",
				"team not found",
				domainErrors.ErrNotFound,
			)
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	// Получаем всех пользователей команды
	users, err := uc.userRepo.GetByTeam(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	// Преобразуем в TeamMembers
	members := make([]entity.TeamMember, 0, len(users))
	for _, user := range users {
		members = append(members, entity.TeamMember{
			UserID:   user.UserID,
			Username: user.Username,
			IsActive: user.IsActive,
		})
	}

	return &entity.TeamWithMembers{
		TeamName: teamName,
		Members:  members,
	}, nil
}
