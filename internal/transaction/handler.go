package transaction

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/response"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type CreditDebitRequest struct {
	Amount      int64  `json:"amount"` // in kobo
	Description string `json:"description"`
}

type TransferRequest struct {
	RecipientEmail string `json:"recipient_email"`
	Amount         int64  `json:"amount"` // in kobo
	Description    string `json:"description"`
}

func (h *Handler) Credit(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreditDebitRequest
	if err := response.ReadJSON(w, r, &req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	record, updatedWallet, err := h.service.Credit(r.Context(), userID, req.Amount, req.Description)
	if err != nil {
		if errors.Is(err, ErrInvalidAmount) {
			_ = response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "failed to credit wallet")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"message":           "wallet credited successfully",
		"transaction":       record,
		"new_balance":       updatedWallet.Balance,
		"balance_formatted": wallet.FormatBalance(updatedWallet.Balance),
	})
}

func (h *Handler) Debit(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreditDebitRequest
	if err := response.ReadJSON(w, r, &req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	record, updatedWallet, err := h.service.Debit(r.Context(), userID, req.Amount, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount):
			_ = response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInsufficientFunds):
			_ = response.Error(w, http.StatusBadRequest, err.Error())
		default:
			_ = response.Error(w, http.StatusInternalServerError, "failed to debit wallet")
		}
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"message":           "wallet debited successfully",
		"transaction":       record,
		"new_balance":       updatedWallet.Balance,
		"balance_formatted": wallet.FormatBalance(updatedWallet.Balance),
	})
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TransferRequest
	if err := response.ReadJSON(w, r, &req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	output, cached, err := h.service.Transfer(r.Context(), userID, req.RecipientEmail, req.Amount, req.Description, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrSelfTransfer):
			_ = response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrRecipientNotFound):
			_ = response.Error(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInsufficientFunds):
			_ = response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrIdempotencyConflict):
			_ = response.Error(w, http.StatusConflict, err.Error())
		default:
			_ = response.Error(w, http.StatusInternalServerError, "failed to process transfer")
		}
		return
	}

	if cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache-Lookup", "HIT")
		w.WriteHeader(cached.StatusCode)
		_, _ = w.Write(cached.Body)
		return
	}

	_ = response.JSON(w, http.StatusOK, output)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page := 1
	pageSize := 20

	query := r.URL.Query()
	if p := query.Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps := query.Get("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	items, total, err := h.service.ListTransactions(r.Context(), userID, page, pageSize)
	if err != nil {
		_ = response.Error(w, http.StatusInternalServerError, "failed to retrieve transactions")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"transactions": items,
		"pagination": response.Envelope{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	txID, err := uuid.Parse(idStr)
	if err != nil {
		_ = response.Error(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	t, err := h.service.GetTransaction(r.Context(), userID, txID)
	if err != nil {
		if errors.Is(err, ErrTransactionNotFound) {
			_ = response.Error(w, http.StatusNotFound, "transaction not found")
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "failed to get transaction")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"transaction": t,
	})
}
