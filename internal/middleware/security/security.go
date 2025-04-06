// Package security provides security middleware for the application.
package security

import (
	"net/http"

	"github.com/hibare/ArguSwarm/internal/constants"
)

// BasicSecurity adds basic security middleware.
func BasicSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Request size limit
		r.Body = http.MaxBytesReader(w, r.Body, constants.DefaultServerRequestSizeLimit)

		next.ServeHTTP(w, r)
	})
}
