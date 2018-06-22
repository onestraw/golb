package balancer

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type Controller struct {
	Address string
	Auth    *Authentication
}

func (c *Controller) Run(service *Service) {
	r := mux.NewRouter()
	r.Handle("/stats", &StatsHandler{service}).Methods("GET")
	r.Handle("/vs/{name}/{action}", ModifyVirtualServerStatus(service)).Methods("POST")
	panic(http.ListenAndServe(c.Address, AuthMiddleware(c.Auth)(r)))
}

type StatsHandler struct {
	service *Service
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result := []string{}
	for _, vs := range h.service.VServers {
		s := fmt.Sprintf("pool-%s:\n%s", vs.Name, vs.stats)
		log.Infof(s)
		result = append(result, s)
	}
	result = append(result, "\n")
	io.WriteString(w, strings.Join(result, "\n"))
}

func ModifyVirtualServerStatus(s *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		action := vars["action"]
		log.Infof("virtual server name %s, action %s", name, action)
		msg := "success"
		var i int
		for i = 0; i < len(s.VServers); i++ {
			if s.VServers[i].Name == name {
				break
			}
		}
		if i < len(s.VServers) {
			vs := s.VServers[i]
			if action == "enable" {
				if vs.Status() == STATUS_ENABLED {
					msg = fmt.Sprintf("%s is already enabled", vs.Name)
				} else {
					vs.Run()
				}
			} else if action == "disable" {
				if vs.Status() == STATUS_DISABLED {
					msg = fmt.Sprintf("%s is already disabled", vs.Name)
				} else {
					vs.Stop()
				}
			} else {
				msg = "unknown action"
			}
		} else {
			msg = fmt.Sprintf("Virtual server [%s] not found", name)
		}
		io.WriteString(w, msg)
	})
}
