package balancer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/config"
)

type Controller struct {
	Address string
	Auth    *Authentication
}

func (c *Controller) Run(service *Service) {
	r := mux.NewRouter()
	r.Handle("/stats", &StatsHandler{service}).Methods("GET")
	r.Handle("/vs/{name}/{action}", ModifyVirtualServerStatus(service)).Methods("POST")
	r.Handle("/vs", AddVirtualServer(service)).Methods("POST")
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

		vs, err := s.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServ err=%v", err)
			WriteError(w, ErrBadRequest)
			return
		}

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

		io.WriteString(w, msg)
	})
}

func AddVirtualServer(s *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var vs config.VirtualServer
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&vs)
		if err != nil {
			log.Errorf("Decode request err=%v", err)
			WriteError(w, ErrBadRequest)
			return
		}

		log.Infof("VirtualServer %v", vs)
		err = s.AddVirtualServer(&vs)
		if err != nil {
			log.Errorf("AddVirtualServ err=%v", err)
			resp := BalancerError{http.StatusBadRequest, err.Error()}
			WriteError(w, resp)
			return
		}

		io.WriteString(w, "Add success")
	})
}
