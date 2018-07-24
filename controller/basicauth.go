package controller

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type authentication struct {
	username string
	password string
}

// BasicAuth is a middleware to check if the request pass the basic auth or not.
func BasicAuth(auth *authentication) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username != auth.username || password != auth.password {
				log.Errorf("Unauthorized (%s:%s) from %s", username, password, r.RemoteAddr)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
