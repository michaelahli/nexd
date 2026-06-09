package middleware

import (
	"net/http"
	"strings"
)

// RequireAdmin restricts access to allowlisted admin emails already present in auth claims.
func RequireAdmin(adminEmails []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		email = strings.TrimSpace(strings.ToLower(email))
		if email != "" {
			allowed[email] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			if len(allowed) == 0 {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			if _, ok := allowed[strings.ToLower(strings.TrimSpace(claims.Email))]; !ok {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
