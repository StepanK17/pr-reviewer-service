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

// UserRepository реализует repository.UserRepository для PostgreSQL
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создает новый репозиторий пользователей
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create создает нового пользователя
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	conn := getConn(ctx, r.pool)

	query := `
		INSERT INTO users (user_id, username, team_name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := conn.Exec(ctx, query,
		user.UserID,
		user.Username,
		user.TeamName,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update обновляет пользователя
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	conn := getConn(ctx, r.pool)

	query := `
		UPDATE users
		SET username = $2, team_name = $3, is_active = $4, updated_at = $5
		WHERE user_id = $1
	`

	result, err := conn.Exec(ctx, query,
		user.UserID,
		user.Username,
		user.TeamName,
		user.IsActive,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domainErrors.ErrNotFound
	}

	return nil
}

// GetByID возвращает пользователя по ID
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*entity.User, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT user_id, username, team_name, is_active, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`

	var user entity.User
	err := conn.QueryRow(ctx, query, userID).Scan(
		&user.UserID,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByTeam возвращает всех пользователей команды
func (r *UserRepository) GetByTeam(ctx context.Context, teamName string) ([]*entity.User, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT user_id, username, team_name, is_active, created_at, updated_at
		FROM users
		WHERE team_name = $1
		ORDER BY username
	`

	rows, err := conn.Query(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by team: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}

// GetActiveByTeam возвращает активных пользователей команды
func (r *UserRepository) GetActiveByTeam(ctx context.Context, teamName string) ([]*entity.User, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT user_id, username, team_name, is_active, created_at, updated_at
		FROM users
		WHERE team_name = $1 AND is_active = true
		ORDER BY username
	`

	rows, err := conn.Query(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users by team: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}

// UpsertBatch создает или обновляет пользователей пакетом
func (r *UserRepository) UpsertBatch(ctx context.Context, users []*entity.User) error {
	conn := getConn(ctx, r.pool)

	query := `
		INSERT INTO users (user_id, username, team_name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
		SET username = EXCLUDED.username,
		    team_name = EXCLUDED.team_name,
		    is_active = EXCLUDED.is_active,
		    updated_at = EXCLUDED.updated_at
	`

	for _, user := range users {
		_, err := conn.Exec(ctx, query,
			user.UserID,
			user.Username,
			user.TeamName,
			user.IsActive,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert user %s: %w", user.UserID, err)
		}
	}

	return nil
}
