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
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secretPass123"`
	FullName string `json:"full_name" example:"Ada Lovelace"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secretPass123"`
}

// Register godoc
// @Summary Register a new user
// @Description Registers a new user and automatically provisions an initial wallet with 0 balance
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} map[string]any
// @Failure 400 {object} map[string]string "Validation error"
// @Failure 409 {object} map[string]string "User with email already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/auth/register [post]
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

// Login godoc
// @Summary User login
// @Description Authenticates a user and returns a signed JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]string "Malformed body"
// @Failure 401 {object} map[string]string "Invalid email or password"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/auth/login [post]
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

// GetProfile godoc
// @Summary Get authenticated user profile
// @Description Returns the current authenticated user's profile
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/users/me [get]
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
