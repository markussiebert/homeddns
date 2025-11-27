package auth

import (
	"encoding/base64"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Config represents authentication configuration
type Config struct {
	Username     string
	PasswordHash string // bcrypt hash
}

// Middleware creates a basic auth middleware
func Middleware(config Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract basic auth credentials
			auth := r.Header.Get("Authorization")
			if auth == "" {
				unauthorized(w)
				return
			}

			// Parse "Basic <base64>"
			const prefix = "Basic "
			if !strings.HasPrefix(auth, prefix) {
				unauthorized(w)
				return
			}

			decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
			if err != nil {
				unauthorized(w)
				return
			}

			// Split username:password
			credentials := string(decoded)
			parts := strings.SplitN(credentials, ":", 2)
			if len(parts) != 2 {
				unauthorized(w)
				return
			}

			username, password := parts[0], parts[1]

			// Verify credentials
			if username != config.Username {
				unauthorized(w)
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(config.PasswordHash), []byte(password)); err != nil {
				unauthorized(w)
				return
			}

			// Authentication successful
			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="DynDNS"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized\n"))
}
