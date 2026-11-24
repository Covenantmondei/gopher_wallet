package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/Covenantmondei/gopher_wallet/internal/response"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				_ = response.Error(w, http.StatusInternalServerError, fmt.Sprintf("%v", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
