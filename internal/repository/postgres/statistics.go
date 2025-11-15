package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
)

// StatisticsRepository реализует repository.StatisticsRepository для PostgreSQL
type StatisticsRepository struct {
	pool *pgxpool.Pool
}

// NewStatisticsRepository создает новый репозиторий статистики
func NewStatisticsRepository(pool *pgxpool.Pool) *StatisticsRepository {
	return &StatisticsRepository{pool: pool}
}

// GetStatistics возвращает общую статистику системы
func (r *StatisticsRepository) GetStatistics(ctx context.Context) (*entity.Statistics, error) {
	stats := &entity.Statistics{
		AssignmentsByUser: make(map[string]int),
		AssignmentsByPR:   make(map[string]int),
	}

	// Получаем общее количество PR
	totalPRsQuery := `SELECT COUNT(*) FROM pull_requests`
	if err := r.pool.QueryRow(ctx, totalPRsQuery).Scan(&stats.TotalPRs); err != nil {
		return nil, fmt.Errorf("failed to get total PRs: %w", err)
	}

	// Получаем количество открытых PR
	openPRsQuery := `SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN'`
	if err := r.pool.QueryRow(ctx, openPRsQuery).Scan(&stats.OpenPRs); err != nil {
		return nil, fmt.Errorf("failed to get open PRs: %w", err)
	}

	// Получаем количество merged PR
	mergedPRsQuery := `SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED'`
	if err := r.pool.QueryRow(ctx, mergedPRsQuery).Scan(&stats.MergedPRs); err != nil {
		return nil, fmt.Errorf("failed to get merged PRs: %w", err)
	}

	// Получаем количество назначений по пользователям
	assignmentsByUserQuery := `
		SELECT u.username, COUNT(pr.reviewer_id) as assignments
		FROM users u
		LEFT JOIN pr_reviewers pr ON u.user_id = pr.reviewer_id
		GROUP BY u.user_id, u.username
		ORDER BY assignments DESC
	`

	rows, err := r.pool.Query(ctx, assignmentsByUserQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var count int
		if err := rows.Scan(&username, &count); err != nil {
			return nil, fmt.Errorf("failed to scan assignment by user: %w", err)
		}
		stats.AssignmentsByUser[username] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate assignments by user: %w", err)
	}

	// Получаем количество назначений по PR
	assignmentsByPRQuery := `
		SELECT p.pull_request_id, COUNT(pr.reviewer_id) as reviewers_count
		FROM pull_requests p
		LEFT JOIN pr_reviewers pr ON p.pull_request_id = pr.pull_request_id
		GROUP BY p.pull_request_id
		ORDER BY reviewers_count DESC
	`

	rows2, err := r.pool.Query(ctx, assignmentsByPRQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments by PR: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var prID string
		var count int
		if err := rows2.Scan(&prID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan assignment by PR: %w", err)
		}
		stats.AssignmentsByPR[prID] = count
	}

	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate assignments by PR: %w", err)
	}

	// Получаем количество команд
	totalTeamsQuery := `SELECT COUNT(*) FROM teams`
	if err := r.pool.QueryRow(ctx, totalTeamsQuery).Scan(&stats.TotalTeams); err != nil {
		return nil, fmt.Errorf("failed to get total teams: %w", err)
	}

	// Получаем количество пользователей
	totalUsersQuery := `SELECT COUNT(*) FROM users`
	if err := r.pool.QueryRow(ctx, totalUsersQuery).Scan(&stats.TotalUsers); err != nil {
		return nil, fmt.Errorf("failed to get total users: %w", err)
	}

	// Получаем количество активных пользователей
	activeUsersQuery := `SELECT COUNT(*) FROM users WHERE is_active = true`
	if err := r.pool.QueryRow(ctx, activeUsersQuery).Scan(&stats.ActiveUsers); err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	return stats, nil
}
