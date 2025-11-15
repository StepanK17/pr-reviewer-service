package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
	domainErrors "github.com/StepanK17/pr-reviewer-service/internal/domain/errors"
)

// PullRequestRepository реализует repository.PullRequestRepository для PostgreSQL
type PullRequestRepository struct {
	pool *pgxpool.Pool
}

// NewPullRequestRepository создает новый репозиторий PR
func NewPullRequestRepository(pool *pgxpool.Pool) *PullRequestRepository {
	return &PullRequestRepository{pool: pool}
}

// Create создает новый PR с ревьюверами
func (r *PullRequestRepository) Create(ctx context.Context, pr *entity.PullRequest) error {
	conn := getConn(ctx, r.pool)

	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := conn.Exec(ctx, query,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.AuthorID,
		pr.Status,
		pr.CreatedAt,
		pr.MergedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	// Добавляем ревьюверов
	if len(pr.AssignedReviewers) > 0 {
		reviewerQuery := `
			INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
			VALUES ($1, $2)
		`

		for _, reviewerID := range pr.AssignedReviewers {
			_, err := conn.Exec(ctx, reviewerQuery, pr.PullRequestID, reviewerID)
			if err != nil {
				return fmt.Errorf("failed to assign reviewer %s: %w", reviewerID, err)
			}
		}
	}

	return nil
}

// Update обновляет PR (статус и ревьюверов)
func (r *PullRequestRepository) Update(ctx context.Context, pr *entity.PullRequest) error {
	conn := getConn(ctx, r.pool)

	// Обновляем PR
	query := `
		UPDATE pull_requests
		SET pull_request_name = $2, status = $3, merged_at = $4
		WHERE pull_request_id = $1
	`

	result, err := conn.Exec(ctx, query,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.Status,
		pr.MergedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domainErrors.ErrNotFound
	}

	// Удаляем старых ревьюверов
	deleteQuery := `DELETE FROM pr_reviewers WHERE pull_request_id = $1`
	_, err = conn.Exec(ctx, deleteQuery, pr.PullRequestID)
	if err != nil {
		return fmt.Errorf("failed to delete old reviewers: %w", err)
	}

	// Добавляем новых ревьюверов
	if len(pr.AssignedReviewers) > 0 {
		reviewerQuery := `
			INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
			VALUES ($1, $2)
		`

		for _, reviewerID := range pr.AssignedReviewers {
			_, err := conn.Exec(ctx, reviewerQuery, pr.PullRequestID, reviewerID)
			if err != nil {
				return fmt.Errorf("failed to assign reviewer %s: %w", reviewerID, err)
			}
		}
	}

	return nil
}

// GetByID возвращает PR по ID
func (r *PullRequestRepository) GetByID(ctx context.Context, prID string) (*entity.PullRequest, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`

	var pr entity.PullRequest
	err := conn.QueryRow(ctx, query, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	reviewersQuery := `
		SELECT reviewer_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
		ORDER BY assigned_at
	`

	rows, err := conn.Query(ctx, reviewersQuery, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, fmt.Errorf("failed to scan reviewer: %w", err)
		}
		reviewers = append(reviewers, reviewerID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate reviewers: %w", err)
	}

	pr.AssignedReviewers = reviewers

	return &pr, nil
}

// GetByReviewer возвращает PRы, где пользователь назначен ревьювером
func (r *PullRequestRepository) GetByReviewer(ctx context.Context, userID string) ([]*entity.PullRequestShort, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT DISTINCT p.pull_request_id, p.pull_request_name, p.author_id, p.status, p.created_at
		FROM pull_requests p
		INNER JOIN pr_reviewers pr ON p.pull_request_id = pr.pull_request_id
		WHERE pr.reviewer_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests by reviewer: %w", err)
	}
	defer rows.Close()

	var prs []*entity.PullRequestShort
	for rows.Next() {
		var pr entity.PullRequestShort
		var createdAt time.Time
		err := rows.Scan(
			&pr.PullRequestID,
			&pr.PullRequestName,
			&pr.AuthorID,
			&pr.Status,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pull request: %w", err)
		}
		prs = append(prs, &pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pull requests: %w", err)
	}

	return prs, nil
}

// Exists проверяет существование PR
func (r *PullRequestRepository) Exists(ctx context.Context, prID string) (bool, error) {
	conn := getConn(ctx, r.pool)

	query := `
		SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)
	`

	var exists bool
	err := conn.QueryRow(ctx, query, prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check pull request existence: %w", err)
	}

	return exists, nil
}

// GetOpenPRsByReviewers возвращает открытые PR для списка ревьюверов
func (r *PullRequestRepository) GetOpenPRsByReviewers(ctx context.Context, reviewerIDs []string) ([]*entity.PullRequest, error) {
	if len(reviewerIDs) == 0 {
		return []*entity.PullRequest{}, nil
	}

	conn := getConn(ctx, r.pool)

	// Используем ANY для поиска по массиву
	query := `
		SELECT DISTINCT p.pull_request_id, p.pull_request_name, p.author_id, p.status, p.created_at, p.merged_at
		FROM pull_requests p
		INNER JOIN pr_reviewers pr ON p.pull_request_id = pr.pull_request_id
		WHERE pr.reviewer_id = ANY($1) AND p.status = 'OPEN'
	`

	rows, err := conn.Query(ctx, query, reviewerIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get open PRs: %w", err)
	}
	defer rows.Close()

	var prs []*entity.PullRequest
	for rows.Next() {
		var pr entity.PullRequest
		err := rows.Scan(
			&pr.PullRequestID,
			&pr.PullRequestName,
			&pr.AuthorID,
			&pr.Status,
			&pr.CreatedAt,
			&pr.MergedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}

		// Получаем ревьюверов для каждого PR
		reviewersQuery := `SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id = $1`
		reviewerRows, err := conn.Query(ctx, reviewersQuery, pr.PullRequestID)
		if err != nil {
			return nil, fmt.Errorf("failed to get reviewers: %w", err)
		}

		var reviewers []string
		for reviewerRows.Next() {
			var reviewerID string
			if err := reviewerRows.Scan(&reviewerID); err != nil {
				reviewerRows.Close()
				return nil, fmt.Errorf("failed to scan reviewer: %w", err)
			}
			reviewers = append(reviewers, reviewerID)
		}
		reviewerRows.Close()

		pr.AssignedReviewers = reviewers
		prs = append(prs, &pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate PRs: %w", err)
	}

	return prs, nil
}
