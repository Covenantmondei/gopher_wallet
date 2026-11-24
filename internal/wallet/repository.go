package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWalletNotFound = errors.New("wallet not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateWithTx(ctx context.Context, tx pgx.Tx, w *Wallet) error {
	query := `
		INSERT INTO wallets (user_id, currency, balance)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := tx.QueryRow(ctx, query, w.UserID, w.Currency, w.Balance).Scan(&w.ID, &w.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert wallet: %w", err)
	}
	return nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Wallet, error) {
	query := `
		SELECT id, user_id, currency, balance, created_at
		FROM wallets
		WHERE user_id = $1
	`
	var w Wallet
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&w.ID,
		&w.UserID,
		&w.Currency,
		&w.Balance,
		&w.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by user id: %w", err)
	}
	return &w, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Wallet, error) {
	query := `
		SELECT id, user_id, currency, balance, created_at
		FROM wallets
		WHERE id = $1
	`
	var w Wallet
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&w.ID,
		&w.UserID,
		&w.Currency,
		&w.Balance,
		&w.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by id: %w", err)
	}
	return &w, nil
}

// GetForUpdate locks the wallet row with SELECT ... FOR UPDATE within a transaction.
func (r *Repository) GetForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*Wallet, error) {
	query := `
		SELECT id, user_id, currency, balance, created_at
		FROM wallets
		WHERE id = $1
		FOR UPDATE
	`
	var w Wallet
	err := tx.QueryRow(ctx, query, id).Scan(
		&w.ID,
		&w.UserID,
		&w.Currency,
		&w.Balance,
		&w.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to lock wallet row: %w", err)
	}
	return &w, nil
}

// UpdateBalance updates the wallet balance within an ongoing transaction.
func (r *Repository) UpdateBalance(ctx context.Context, tx pgx.Tx, id uuid.UUID, newBalance int64) error {
	query := `
		UPDATE wallets
		SET balance = $1
		WHERE id = $2
	`
	tag, err := tx.Exec(ctx, query, newBalance, id)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWalletNotFound
	}
	return nil
}
