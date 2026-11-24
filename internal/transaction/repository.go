package transaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrIdempotencyNotFound = errors.New("idempotency record not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateWithTx(ctx context.Context, tx pgx.Tx, t *Transaction) error {
	query := `
		INSERT INTO transactions (reference, wallet_id, type, amount, status, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := tx.QueryRow(ctx, query,
		t.Reference,
		t.WalletID,
		t.Type,
		t.Amount,
		t.Status,
		t.Description,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}
	return nil
}

func (r *Repository) CreateTransferWithTx(ctx context.Context, tx pgx.Tx, tr *Transfer) error {
	query := `
		INSERT INTO transfers (source_wallet_id, target_wallet_id, debit_transaction_id, credit_transaction_id, amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := tx.QueryRow(ctx, query,
		tr.SourceWalletID,
		tr.TargetWalletID,
		tr.DebitTransactionID,
		tr.CreditTransactionID,
		tr.Amount,
	).Scan(&tr.ID, &tr.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert transfer: %w", err)
	}
	return nil
}

func (r *Repository) GetByIDAndWalletID(ctx context.Context, id uuid.UUID, walletID uuid.UUID) (*Transaction, error) {
	query := `
		SELECT id, reference, wallet_id, type, amount, status, description, created_at
		FROM transactions
		WHERE id = $1 AND wallet_id = $2
	`
	var t Transaction
	err := r.pool.QueryRow(ctx, query, id, walletID).Scan(
		&t.ID,
		&t.Reference,
		&t.WalletID,
		&t.Type,
		&t.Amount,
		&t.Status,
		&t.Description,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return &t, nil
}

func (r *Repository) ListByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]Transaction, int, error) {
	countQuery := `SELECT COUNT(*) FROM transactions WHERE wallet_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, walletID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	query := `
		SELECT id, reference, wallet_id, type, amount, status, description, created_at
		FROM transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, walletID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var list []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.ID,
			&t.Reference,
			&t.WalletID,
			&t.Type,
			&t.Amount,
			&t.Status,
			&t.Description,
			&t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction: %w", err)
		}
		list = append(list, t)
	}

	return list, total, nil
}

func (r *Repository) GetIdempotencyRecord(ctx context.Context, userID uuid.UUID, key string) (*IdempotencyRecord, error) {
	query := `
		SELECT id, key, user_id, request_hash, response_code, response_body, created_at
		FROM idempotency_keys
		WHERE user_id = $1 AND key = $2
	`
	var rec IdempotencyRecord
	err := r.pool.QueryRow(ctx, query, userID, key).Scan(
		&rec.ID,
		&rec.Key,
		&rec.UserID,
		&rec.RequestHash,
		&rec.ResponseCode,
		&rec.ResponseBody,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIdempotencyNotFound
		}
		return nil, fmt.Errorf("failed to get idempotency key: %w", err)
	}
	return &rec, nil
}

func (r *Repository) SaveIdempotencyRecordWithTx(ctx context.Context, tx pgx.Tx, rec *IdempotencyRecord) error {
	query := `
		INSERT INTO idempotency_keys (key, user_id, request_hash, response_code, response_body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := tx.QueryRow(ctx, query,
		rec.Key,
		rec.UserID,
		rec.RequestHash,
		rec.ResponseCode,
		rec.ResponseBody,
	).Scan(&rec.ID, &rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save idempotency record: %w", err)
	}
	return nil
}
