package middleware

import (
	"log"
	"net/http"
)

// Recovery converts panics into HTTP 500 responses and keeps the server alive.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("http panic request_id=%s method=%s path=%s error=%v", RequestIDFromContext(r.Context()), r.Method, r.URL.Path, recovered)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
