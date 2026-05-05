package httpx

import (
	"log"
	"net/http"
	"runtime/debug"
)

func RecoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic: %v\n%s", recovered, debug.Stack())
				WriteError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
