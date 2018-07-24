package controller

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// Authentication holds username/password.
type Authentication struct {
	Username string
	Password string
}

// BasicAuth is a middleware to check if the request pass the basic auth or not.
func BasicAuth(auth *Authentication) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username != auth.Username || password != auth.Password {
				log.Errorf("Unauthorized (%s:%s) from %s", username, password, r.RemoteAddr)
				WriteError(w, ErrUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
