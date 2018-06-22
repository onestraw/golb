package balancer

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Authentication struct {
	Username string
	Password string
}

func AuthMiddleware(auth *Authentication) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			username := r.Header.Get("Username")
			password := r.Header.Get("Password")
			if username == auth.Username && password == auth.Password {
				next.ServeHTTP(w, r)
			} else {
				log.Errorf("Unauthorized <%s, %s> from %s", username, password, r.RemoteAddr)
				WriteError(w, ErrUnauthorized)
				return
			}
		}
		return http.HandlerFunc(fn)
	}
}
