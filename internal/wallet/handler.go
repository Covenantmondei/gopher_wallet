package wallet

import (
	"errors"
	"net/http"

	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	wallet, err := h.service.GetWalletByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrWalletNotFound) {
			_ = response.Error(w, http.StatusNotFound, "wallet not found")
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "failed to retrieve wallet")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"wallet":            wallet,
		"balance_formatted": FormatBalance(wallet.Balance),
	})
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	wallet, err := h.service.GetWalletByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrWalletNotFound) {
			_ = response.Error(w, http.StatusNotFound, "wallet not found")
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "failed to retrieve balance")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"currency":          wallet.Currency,
		"balance":           wallet.Balance,
		"balance_formatted": FormatBalance(wallet.Balance),
	})
}
