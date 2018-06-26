package balancer

import (
	"encoding/base64"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Authentication struct {
	Username string
	Password string
}

func AuthMiddleware(auth *Authentication) func(http.Handler) http.Handler {
	validate := func(username, password string) bool {
		if username == auth.Username && password == auth.Password {
			return true
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(auth) != 2 || auth[0] != "Basic" {
				WriteError(w, ErrUnauthorized)
				return
			}

			payload, _ := base64.StdEncoding.DecodeString(auth[1])
			pair := strings.SplitN(string(payload), ":", 2)
			if len(pair) != 2 || !validate(pair[0], pair[1]) {
				log.Errorf("Unauthorized %v from %s", pair, r.RemoteAddr)
				WriteError(w, ErrUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
