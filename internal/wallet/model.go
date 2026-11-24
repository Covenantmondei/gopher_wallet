package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Currency  string    `json:"currency"`
	Balance   int64     `json:"balance"` // Stored in kobo
	CreatedAt time.Time `json:"created_at"`
}
