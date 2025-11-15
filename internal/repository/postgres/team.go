package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
	domainErrors "github.com/StepanK17/pr-reviewer-service/internal/domain/errors"
)

// TeamRepository реализует repository.TeamRepository для PostgreSQL
type TeamRepository struct {
	pool *pgxpool.Pool
}

// NewTeamRepository создает новый репозиторий команд
func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

// Create создает новую команду
func (r *TeamRepository) Create(ctx context.Context, team *entity.Team) error {
	conn := getConn(ctx, r.pool)

	query := `
		INSERT INTO teams (team_name, created_at, updated_at)
		VALUES ($1, $2, $3)
	`

	_, err := conn.Exec(ctx, query, team.TeamName, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	return nil
}

// GetByName возвращает команду по имени
func (r *TeamRepository) GetByName(ctx context.Context, teamName string) (*entity.Team, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT team_name, created_at, updated_at
		FROM teams
		WHERE team_name = $1
	`

	var team entity.Team
	err := conn.QueryRow(ctx, query, teamName).Scan(
		&team.TeamName,
		&team.CreatedAt,
		&team.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return &team, nil
}

// Exists проверяет существование команды
func (r *TeamRepository) Exists(ctx context.Context, teamName string) (bool, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)
	`

	var exists bool
	err := conn.QueryRow(ctx, query, teamName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check team existence: %w", err)
	}

	return exists, nil
}
