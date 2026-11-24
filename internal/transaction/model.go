package transaction

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeCredit Type = "CREDIT"
	TypeDebit  Type = "DEBIT"
)

type Status string

const (
	StatusPending Status = "PENDING"
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
)

type Transaction struct {
	ID          uuid.UUID `json:"id"`
	Reference   string    `json:"reference"`
	WalletID    uuid.UUID `json:"wallet_id"`
	Type        Type      `json:"type"`
	Amount      int64     `json:"amount"` // in kobo
	Status      Status    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Transfer struct {
	ID                 uuid.UUID `json:"id"`
	SourceWalletID     uuid.UUID `json:"source_wallet_id"`
	TargetWalletID     uuid.UUID `json:"target_wallet_id"`
	DebitTransactionID uuid.UUID `json:"debit_transaction_id"`
	CreditTransactionID uuid.UUID `json:"credit_transaction_id"`
	Amount             int64     `json:"amount"` // in kobo
	CreatedAt          time.Time `json:"created_at"`
}

type IdempotencyRecord struct {
	ID           uuid.UUID `json:"id"`
	Key          string    `json:"key"`
	UserID       uuid.UUID `json:"user_id"`
	RequestHash  string    `json:"request_hash"`
	ResponseCode int       `json:"response_code"`
	ResponseBody []byte    `json:"response_body"`
	CreatedAt    time.Time `json:"created_at"`
}
