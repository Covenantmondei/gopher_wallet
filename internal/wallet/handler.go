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

// GetWallet godoc
// @Summary Get user wallet details
// @Description Returns the authenticated user's wallet with balance
// @Tags Wallet
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Wallet not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/wallet [get]
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

// GetBalance godoc
// @Summary Get user wallet balance
// @Description Returns the authenticated user's wallet balance in kobo and formatted NGN
// @Tags Wallet
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Wallet not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/wallet/balance [get]
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
