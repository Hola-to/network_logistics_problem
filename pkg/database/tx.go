package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxFunc функция, выполняемая в транзакции
type TxFunc func(tx pgx.Tx) error

// WithTransaction выполняет функцию в транзакции
func WithTransaction(ctx context.Context, db DB, fn TxFunc) error {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx) //nolint:errcheck // best effort on panic
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionResult выполняет функцию в транзакции с возвратом результата
func WithTransactionResult[T any](ctx context.Context, db DB, fn func(tx pgx.Tx) (T, error)) (T, error) {
	var result T

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx) //nolint:errcheck // best effort on panic
			panic(p)
		}
	}()

	result, err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return result, fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return result, err
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}
