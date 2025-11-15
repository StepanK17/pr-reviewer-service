package repository

import (
	"context"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, userID string) (*entity.User, error)
	GetByTeam(ctx context.Context, teamName string) ([]*entity.User, error)
	GetActiveByTeam(ctx context.Context, teamName string) ([]*entity.User, error)
	UpsertBatch(ctx context.Context, users []*entity.User) error
}

type TeamRepository interface {
	Create(ctx context.Context, team *entity.Team) error
	GetByName(ctx context.Context, teamName string) (*entity.Team, error)
	Exists(ctx context.Context, teamName string) (bool, error)
}

type PullRequestRepository interface {
	Create(ctx context.Context, pr *entity.PullRequest) error
	Update(ctx context.Context, pr *entity.PullRequest) error
	GetByID(ctx context.Context, prID string) (*entity.PullRequest, error)
	GetByReviewer(ctx context.Context, userID string) ([]*entity.PullRequestShort, error)
	Exists(ctx context.Context, prID string) (bool, error)
	GetOpenPRsByReviewers(ctx context.Context, reviewerIDs []string) ([]*entity.PullRequest, error)
}

type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type StatisticsRepository interface {
	GetStatistics(ctx context.Context) (*entity.Statistics, error)
}
