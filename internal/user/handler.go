package user

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

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := response.ReadJSON(w, r, &req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, wallet, err := h.service.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateEmail):
			_ = response.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrPasswordTooWeak), errors.Is(err, ErrNameRequired):
			_ = response.Error(w, http.StatusBadRequest, err.Error())
		default:
			_ = response.Error(w, http.StatusInternalServerError, "failed to register user")
		}
		return
	}

	_ = response.JSON(w, http.StatusCreated, response.Envelope{
		"message": "user registered successfully",
		"user":    user,
		"wallet":  wallet,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.ReadJSON(w, r, &req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, token, expiresAt, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			_ = response.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "failed to process login")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"message":    "login successful",
		"token":      token,
		"expires_at": expiresAt,
		"user":       user,
	})
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = response.Error(w, http.StatusNotFound, "user profile not found")
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "failed to load profile")
		return
	}

	_ = response.JSON(w, http.StatusOK, response.Envelope{
		"user": user,
	})
}
