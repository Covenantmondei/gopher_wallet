package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDuplicateEmail = errors.New("user with this email already exists")
	ErrUserNotFound   = errors.New("user not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateWithTx(ctx context.Context, tx pgx.Tx, u *User) error {
	query := `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := tx.QueryRow(ctx, query, strings.ToLower(u.Email), u.PasswordHash, u.FullName).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, created_at
		FROM users
		WHERE email = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, strings.ToLower(email)).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, created_at
		FROM users
		WHERE id = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &u, nil
}
