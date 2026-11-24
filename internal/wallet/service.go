package wallet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*Wallet, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func FormatBalance(kobo int64) string {
	naira := float64(kobo) / 100.0
	return fmt.Sprintf("%.2f", naira)
}
