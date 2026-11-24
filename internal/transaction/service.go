package transaction

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Covenantmondei/gopher_wallet/internal/user"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidAmount         = errors.New("amount must be greater than zero")
	ErrInsufficientFunds     = errors.New("insufficient funds in wallet")
	ErrSelfTransfer          = errors.New("cannot transfer funds to your own wallet")
	ErrRecipientNotFound     = errors.New("recipient user not found")
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different request payload")
	ErrRecipientWalletMissing = errors.New("recipient wallet not found")
)

type CachedTransferResult struct {
	StatusCode int
	Body       []byte
}

type Service struct {
	pool            *pgxpool.Pool
	txRepo          *Repository
	walletRepo      *wallet.Repository
	userRepo        *user.Repository
}

func NewService(
	pool *pgxpool.Pool,
	txRepo *Repository,
	walletRepo *wallet.Repository,
	userRepo *user.Repository,
) *Service {
	return &Service{
		pool:       pool,
		txRepo:     txRepo,
		walletRepo: walletRepo,
		userRepo:   userRepo,
	}
}

func (s *Service) Credit(ctx context.Context, userID uuid.UUID, amount int64, description string) (*Transaction, *wallet.Wallet, error) {
	if amount <= 0 {
		return nil, nil, ErrInvalidAmount
	}
	if description == "" {
		description = "Wallet credit deposit"
	}

	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Acquire row-level lock
	lockedWallet, err := s.walletRepo.GetForUpdate(ctx, tx, w.ID)
	if err != nil {
		return nil, nil, err
	}

	newBalance := lockedWallet.Balance + amount
	if err := s.walletRepo.UpdateBalance(ctx, tx, lockedWallet.ID, newBalance); err != nil {
		return nil, nil, err
	}

	record := &Transaction{
		Reference:   fmt.Sprintf("CR-%s", uuid.New().String()[:12]),
		WalletID:    lockedWallet.ID,
		Type:        TypeCredit,
		Amount:      amount,
		Status:      StatusSuccess,
		Description: description,
	}

	if err := s.txRepo.CreateWithTx(ctx, tx, record); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit credit transaction: %w", err)
	}

	lockedWallet.Balance = newBalance
	return record, lockedWallet, nil
}

func (s *Service) Debit(ctx context.Context, userID uuid.UUID, amount int64, description string) (*Transaction, *wallet.Wallet, error) {
	if amount <= 0 {
		return nil, nil, ErrInvalidAmount
	}
	if description == "" {
		description = "Wallet debit withdrawal"
	}

	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Acquire row-level lock
	lockedWallet, err := s.walletRepo.GetForUpdate(ctx, tx, w.ID)
	if err != nil {
		return nil, nil, err
	}

	if lockedWallet.Balance < amount {
		return nil, nil, ErrInsufficientFunds
	}

	newBalance := lockedWallet.Balance - amount
	if err := s.walletRepo.UpdateBalance(ctx, tx, lockedWallet.ID, newBalance); err != nil {
		return nil, nil, err
	}

	record := &Transaction{
		Reference:   fmt.Sprintf("DB-%s", uuid.New().String()[:12]),
		WalletID:    lockedWallet.ID,
		Type:        TypeDebit,
		Amount:      amount,
		Status:      StatusSuccess,
		Description: description,
	}

	if err := s.txRepo.CreateWithTx(ctx, tx, record); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit debit transaction: %w", err)
	}

	lockedWallet.Balance = newBalance
	return record, lockedWallet, nil
}

type TransferOutput struct {
	Transfer          *Transfer `json:"transfer"`
	SenderTransaction *Transaction `json:"sender_transaction"`
	SenderBalance     int64     `json:"sender_balance"`
	Message           string    `json:"message"`
}

func (s *Service) Transfer(
	ctx context.Context,
	senderUserID uuid.UUID,
	recipientEmail string,
	amount int64,
	description string,
	idempotencyKey string,
) (*TransferOutput, *CachedTransferResult, error) {
	if amount <= 0 {
		return nil, nil, ErrInvalidAmount
	}
	if description == "" {
		description = "P2P transfer"
	}

	// 1. Check idempotency if key is present
	var reqHash string
	if idempotencyKey != "" {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%s", senderUserID.String(), strings.ToLower(recipientEmail), amount, description)))
		reqHash = hex.EncodeToString(sum[:])

		existing, err := s.txRepo.GetIdempotencyRecord(ctx, senderUserID, idempotencyKey)
		if err == nil {
			if existing.RequestHash != reqHash {
				return nil, nil, ErrIdempotencyConflict
			}
			// Return cached response directly
			return nil, &CachedTransferResult{
				StatusCode: existing.ResponseCode,
				Body:       existing.ResponseBody,
			}, nil
		} else if !errors.Is(err, ErrIdempotencyNotFound) {
			return nil, nil, err
		}
	}

	// 2. Resolve sender wallet
	senderWallet, err := s.walletRepo.GetByUserID(ctx, senderUserID)
	if err != nil {
		return nil, nil, err
	}

	// 3. Resolve recipient user & wallet
	recipientUser, err := s.userRepo.GetByEmail(ctx, recipientEmail)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, nil, ErrRecipientNotFound
		}
		return nil, nil, err
	}

	if recipientUser.ID == senderUserID {
		return nil, nil, ErrSelfTransfer
	}

	recipientWallet, err := s.walletRepo.GetByUserID(ctx, recipientUser.ID)
	if err != nil {
		if errors.Is(err, wallet.ErrWalletNotFound) {
			return nil, nil, ErrRecipientWalletMissing
		}
		return nil, nil, err
	}

	// 4. Begin single atomic transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transfer transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 5. Deterministic lock acquisition to prevent deadlocks:
	// Always lock the wallet with the smaller UUID bytes first.
	var firstID, secondID uuid.UUID
	if bytes.Compare(senderWallet.ID[:], recipientWallet.ID[:]) < 0 {
		firstID = senderWallet.ID
		secondID = recipientWallet.ID
	} else {
		firstID = recipientWallet.ID
		secondID = senderWallet.ID
	}

	lockedFirst, err := s.walletRepo.GetForUpdate(ctx, tx, firstID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lock wallet %s: %w", firstID, err)
	}

	lockedSecond, err := s.walletRepo.GetForUpdate(ctx, tx, secondID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lock wallet %s: %w", secondID, err)
	}

	// Assign locked state back to sender & recipient
	var lockedSender, lockedRecipient *wallet.Wallet
	if lockedFirst.ID == senderWallet.ID {
		lockedSender = lockedFirst
		lockedRecipient = lockedSecond
	} else {
		lockedSender = lockedSecond
		lockedRecipient = lockedFirst
	}

	// 6. Validate balance
	if lockedSender.Balance < amount {
		return nil, nil, ErrInsufficientFunds
	}

	// 7. Update balances
	newSenderBalance := lockedSender.Balance - amount
	newRecipientBalance := lockedRecipient.Balance + amount

	if err := s.walletRepo.UpdateBalance(ctx, tx, lockedSender.ID, newSenderBalance); err != nil {
		return nil, nil, err
	}
	if err := s.walletRepo.UpdateBalance(ctx, tx, lockedRecipient.ID, newRecipientBalance); err != nil {
		return nil, nil, err
	}

	// 8. Create debit and credit transaction records
	transferRef := uuid.New().String()[:12]
	debitTx := &Transaction{
		Reference:   fmt.Sprintf("TR-DB-%s", transferRef),
		WalletID:    lockedSender.ID,
		Type:        TypeDebit,
		Amount:      amount,
		Status:      StatusSuccess,
		Description: fmt.Sprintf("Transfer to %s: %s", recipientEmail, description),
	}
	if err := s.txRepo.CreateWithTx(ctx, tx, debitTx); err != nil {
		return nil, nil, err
	}

	creditTx := &Transaction{
		Reference:   fmt.Sprintf("TR-CR-%s", transferRef),
		WalletID:    lockedRecipient.ID,
		Type:        TypeCredit,
		Amount:      amount,
		Status:      StatusSuccess,
		Description: fmt.Sprintf("Transfer received: %s", description),
	}
	if err := s.txRepo.CreateWithTx(ctx, tx, creditTx); err != nil {
		return nil, nil, err
	}

	// 9. Create transfer audit record
	transferRecord := &Transfer{
		SourceWalletID:      lockedSender.ID,
		TargetWalletID:      lockedRecipient.ID,
		DebitTransactionID:  debitTx.ID,
		CreditTransactionID: creditTx.ID,
		Amount:              amount,
	}
	if err := s.txRepo.CreateTransferWithTx(ctx, tx, transferRecord); err != nil {
		return nil, nil, err
	}

	output := &TransferOutput{
		Transfer:          transferRecord,
		SenderTransaction: debitTx,
		SenderBalance:     newSenderBalance,
		Message:           "transfer completed successfully",
	}

	// 10. Record idempotency inside same transaction if key provided
	if idempotencyKey != "" {
		bodyBytes, err := json.Marshal(output)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal idempotency body: %w", err)
		}

		idempRecord := &IdempotencyRecord{
			Key:          idempotencyKey,
			UserID:       senderUserID,
			RequestHash:  reqHash,
			ResponseCode: 200,
			ResponseBody: bodyBytes,
		}
		if err := s.txRepo.SaveIdempotencyRecordWithTx(ctx, tx, idempRecord); err != nil {
			return nil, nil, err
		}
	}

	// 11. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transfer: %w", err)
	}

	return output, nil, nil
}

func (s *Service) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]Transaction, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return s.txRepo.ListByWalletID(ctx, w.ID, pageSize, offset)
}

func (s *Service) GetTransaction(ctx context.Context, userID uuid.UUID, txID uuid.UUID) (*Transaction, error) {
	w, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.txRepo.GetByIDAndWalletID(ctx, txID, w.ID)
}
