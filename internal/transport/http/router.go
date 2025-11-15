package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/handler"
	customMiddleware "github.com/StepanK17/pr-reviewer-service/internal/transport/http/middleware"
)

// RouterConfig содержит конфигурацию для роутера
type RouterConfig struct {
	TeamHandler        *handler.TeamHandler
	UserHandler        *handler.UserHandler
	PullRequestHandler *handler.PullRequestHandler
	HealthHandler      *handler.HealthHandler
	StatisticsHandler  *handler.StatisticsHandler
	AdminToken         string
}

// NewRouter создает и настраивает роутер
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Health check
	r.Get("/health", cfg.HealthHandler.Check)

	// Statistics
	r.Get("/statistics", cfg.StatisticsHandler.GetStatistics)

	// Teams
	r.Post("/team/add", cfg.TeamHandler.CreateTeam)
	r.Get("/team/get", cfg.TeamHandler.GetTeam)
	r.Post("/team/add", cfg.TeamHandler.CreateTeam)
	r.Get("/team/get", cfg.TeamHandler.GetTeam)
	r.With(customMiddleware.AdminAuth(cfg.AdminToken)).Post("/team/deactivateMembers", cfg.TeamHandler.DeactivateTeamMembers)

	// Users
	r.With(customMiddleware.AdminAuth(cfg.AdminToken)).Post("/users/setIsActive", cfg.UserHandler.SetIsActive)
	r.Get("/users/getReview", cfg.UserHandler.GetReview)

	// Pull Requests
	r.Post("/pullRequest/create", cfg.PullRequestHandler.CreatePR)
	r.Post("/pullRequest/merge", cfg.PullRequestHandler.MergePR)
	r.Post("/pullRequest/reassign", cfg.PullRequestHandler.ReassignReviewer)

	return r
}
