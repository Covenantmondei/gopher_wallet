package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Covenantmondei/gopher_wallet/internal/response"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
	EmailContextKey  contextKey = "email"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uuid.UUID, email string, secret string, expirationHours int) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(expirationHours) * time.Hour)
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gopherwallet",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				_ = response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				_ = response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenString := parts[1]
			claims := &Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				_ = response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailContextKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(UserIDContextKey)
	id, ok := val.(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}
