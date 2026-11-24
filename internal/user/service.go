package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail       = errors.New("a valid email address is required")
	ErrPasswordTooWeak    = errors.New("password must be at least 8 characters long")
	ErrNameRequired       = errors.New("full name is required")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type Service struct {
	pool               *pgxpool.Pool
	userRepo           *Repository
	walletRepo         *wallet.Repository
	jwtSecret          string
	jwtExpirationHours int
}

func NewService(
	pool *pgxpool.Pool,
	userRepo *Repository,
	walletRepo *wallet.Repository,
	jwtSecret string,
	jwtExpirationHours int,
) *Service {
	return &Service{
		pool:               pool,
		userRepo:           userRepo,
		walletRepo:         walletRepo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

func (s *Service) Register(ctx context.Context, email, password, fullName string) (*User, *wallet.Wallet, error) {
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)

	if email == "" || !strings.Contains(email, "@") {
		return nil, nil, ErrInvalidEmail
	}
	if len(password) < 8 {
		return nil, nil, ErrPasswordTooWeak
	}
	if fullName == "" {
		return nil, nil, ErrNameRequired
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	newUser := &User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
	}

	if err := s.userRepo.CreateWithTx(ctx, tx, newUser); err != nil {
		return nil, nil, err
	}

	newWallet := &wallet.Wallet{
		UserID:   newUser.ID,
		Currency: "NGN",
		Balance:  0,
	}

	if err := s.walletRepo.CreateWithTx(ctx, tx, newWallet); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit registration transaction: %w", err)
	}

	return newUser, newWallet, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*User, string, time.Time, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, "", time.Time{}, ErrInvalidCredentials
		}
		return nil, "", time.Time{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	token, expiresAt, err := middleware.GenerateToken(user.ID, user.Email, s.jwtSecret, s.jwtExpirationHours)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, expiresAt, nil
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.userRepo.GetByID(ctx, userID)
}
