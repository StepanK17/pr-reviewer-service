package dto

import (
	"time"

	"github.com/StepanK17/pr-reviewer-service/internal/domain/entity"
)

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail содержит детали ошибки
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TeamMemberDTO представляет участника команды
type TeamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// TeamDTO представляет команду
type TeamDTO struct {
	TeamName string          `json:"team_name"`
	Members  []TeamMemberDTO `json:"members"`
}

// CreateTeamRequest запрос на создание команды
type CreateTeamRequest struct {
	TeamName string          `json:"team_name"`
	Members  []TeamMemberDTO `json:"members"`
}

// CreateTeamResponse ответ на создание команды
type CreateTeamResponse struct {
	Team TeamDTO `json:"team"`
}

// UserDTO представляет пользователя
type UserDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

// SetIsActiveRequest запрос на изменение активности пользователя
type SetIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

// SetIsActiveResponse ответ на изменение активности
type SetIsActiveResponse struct {
	User UserDTO `json:"user"`
}

// PullRequestDTO представляет Pull Request
type PullRequestDTO struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt         *string  `json:"createdAt,omitempty"`
	MergedAt          *string  `json:"mergedAt,omitempty"`
}

// PullRequestShortDTO представляет краткую информацию о PR
type PullRequestShortDTO struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

// CreatePRRequest запрос на создание PR
type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

// CreatePRResponse ответ на создание PR
type CreatePRResponse struct {
	PR PullRequestDTO `json:"pr"`
}

// MergePRRequest запрос на мёрдж PR
type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

// MergePRResponse ответ на мёрдж PR
type MergePRResponse struct {
	PR PullRequestDTO `json:"pr"`
}

// ReassignRequest запрос на переназначение ревьювера
type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

// ReassignResponse ответ на переназначение
type ReassignResponse struct {
	PR         PullRequestDTO `json:"pr"`
	ReplacedBy string         `json:"replaced_by"`
}

// GetUserReviewsResponse ответ на получение PR пользователя
type GetUserReviewsResponse struct {
	UserID       string                `json:"user_id"`
	PullRequests []PullRequestShortDTO `json:"pull_requests"`
}

// Маппинг функции

// ToTeamDTO преобразует entity в DTO
func ToTeamDTO(team *entity.TeamWithMembers) TeamDTO {
	members := make([]TeamMemberDTO, 0, len(team.Members))
	for _, m := range team.Members {
		members = append(members, TeamMemberDTO{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	return TeamDTO{
		TeamName: team.TeamName,
		Members:  members,
	}
}

// ToTeamEntity преобразует DTO в entity
func ToTeamEntity(dto *CreateTeamRequest) *entity.TeamWithMembers {
	members := make([]entity.TeamMember, 0, len(dto.Members))
	for _, m := range dto.Members {
		members = append(members, entity.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	return &entity.TeamWithMembers{
		TeamName: dto.TeamName,
		Members:  members,
	}
}

// ToUserDTO преобразует entity в DTO
func ToUserDTO(user *entity.User) UserDTO {
	return UserDTO{
		UserID:   user.UserID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

// ToPullRequestDTO преобразует entity в DTO
func ToPullRequestDTO(pr *entity.PullRequest) PullRequestDTO {
	dto := PullRequestDTO{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
	}

	// Форматируем время в RFC3339
	if !pr.CreatedAt.IsZero() {
		createdAt := pr.CreatedAt.Format(time.RFC3339)
		dto.CreatedAt = &createdAt
	}

	if pr.MergedAt != nil && !pr.MergedAt.IsZero() {
		mergedAt := pr.MergedAt.Format(time.RFC3339)
		dto.MergedAt = &mergedAt
	}

	return dto
}

// ToPullRequestShortDTO преобразует entity в short DTO
func ToPullRequestShortDTO(pr *entity.PullRequestShort) PullRequestShortDTO {
	return PullRequestShortDTO{
		PullRequestID:   pr.PullRequestID,
		PullRequestName: pr.PullRequestName,
		AuthorID:        pr.AuthorID,
		Status:          string(pr.Status),
	}
}

// ToPullRequestShortDTOs преобразует список entities в список DTOs
func ToPullRequestShortDTOs(prs []*entity.PullRequestShort) []PullRequestShortDTO {
	dtos := make([]PullRequestShortDTO, 0, len(prs))
	for _, pr := range prs {
		dtos = append(dtos, ToPullRequestShortDTO(pr))
	}
	return dtos
}

type DeactivateTeamMembersRequest struct {
	TeamName string `json:"team_name"`
}

// DeactivateTeamMembersResponse ответ на массовую деактивацию
type DeactivateTeamMembersResponse struct {
	DeactivatedCount int      `json:"deactivated_count"`
	ReassignedPRs    int      `json:"reassigned_prs"`
	UserIDs          []string `json:"user_ids"`
}
