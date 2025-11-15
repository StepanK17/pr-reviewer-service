package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// TransactionManager реализует управление транзакциями
type TransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager создает новый менеджер транзакций
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// RunInTransaction выполняет функцию в транзакции
func (tm *TransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Если транзакция уже существует в контексте, используем её
	if tx := extractTx(ctx); tx != nil {
		return fn(ctx)
	}

	// Начинаем новую транзакцию
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Сохраняем транзакцию в контекст
	ctx = injectTx(ctx, tx)

	// Выполняем функцию
	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// injectTx добавляет транзакцию в контекст
func injectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// extractTx извлекает транзакцию из контекста
func extractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// getConn возвращает соединение или траынзакцию из контекста
func getConn(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx := extractTx(ctx); tx != nil {
		return tx
	}
	return pool
}

// querier интерфейс для выполнения запросов (поддерживает и Pool, и Tx)
type querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}
