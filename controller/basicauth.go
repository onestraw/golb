package controller

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Authentication struct {
	Username string
	Password string
}

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
