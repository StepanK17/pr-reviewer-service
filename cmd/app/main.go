package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StepanK17/pr-reviewer-service/internal/config"
	"github.com/StepanK17/pr-reviewer-service/internal/repository/postgres"
	httpTransport "github.com/StepanK17/pr-reviewer-service/internal/transport/http"
	"github.com/StepanK17/pr-reviewer-service/internal/transport/http/handler"
	"github.com/StepanK17/pr-reviewer-service/internal/usecase"
)

func main() {
	//Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключаемся к базе данных
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to database")

	// Применяем миграции
	if err := runMigrations(cfg.GetDSN()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations applied successfully")

	// Инициализируем репозитории
	teamRepo := postgres.NewTeamRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	prRepo := postgres.NewPullRequestRepository(pool)
	statsRepo := postgres.NewStatisticsRepository(pool)
	txManager := postgres.NewTransactionManager(pool)

	// Инициализируем use cases
	teamUseCase := usecase.NewTeamUseCase(teamRepo, userRepo, txManager, prRepo)
	userUseCase := usecase.NewUserUseCase(userRepo, prRepo)
	prUseCase := usecase.NewPullRequestUseCase(prRepo, userRepo, txManager)
	statsUseCase := usecase.NewStatisticsUseCase(statsRepo)

	// Инициализируем handlers
	teamHandler := handler.NewTeamHandler(teamUseCase)
	userHandler := handler.NewUserHandler(userUseCase)
	prHandler := handler.NewPullRequestHandler(prUseCase)
	healthHandler := handler.NewHealthHandler()
	statsHandler := handler.NewStatisticsHandler(statsUseCase)

	// Создаем роутер
	router := httpTransport.NewRouter(httpTransport.RouterConfig{
		TeamHandler:        teamHandler,
		UserHandler:        userHandler,
		PullRequestHandler: prHandler,
		HealthHandler:      healthHandler,
		StatisticsHandler:  statsHandler,
		AdminToken:         cfg.AdminToken,
	})

	// Создаем HTTP сервер
	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Применяем миграции базы данных
func runMigrations(dsn string) error {
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
