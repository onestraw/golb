package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/config"
)

type Controller struct {
	Address string
	Auth    *Authentication
}

func New(ctlCfg *config.Controller) *Controller {
	return &Controller{
		Address: ctlCfg.Address,
		Auth:    &Authentication{ctlCfg.Auth.Username, ctlCfg.Auth.Password},
	}
}

func (c *Controller) Run(balancer *balancer.Balancer) {
	r := mux.NewRouter()
	r.Handle("/stats", &StatsHandler{balancer}).Methods("GET")
	r.Handle("/vs", AddVirtualServer(balancer)).Methods("POST")
	r.Handle("/vs", ListAllVirtualServer(balancer)).Methods("GET")
	r.Handle("/vs/{name}", ModifyVirtualServerStatus(balancer)).Methods("POST")
	r.Handle("/vs/{name}", ListVirtualServer(balancer)).Methods("GET")
	r.Handle("/vs/{name}/pool", AddPoolMember(balancer)).Methods("POST")
	r.Handle("/vs/{name}/pool", DeletePoolMember(balancer)).Methods("DELETE")
	go func() {
		if err := http.ListenAndServe(c.Address, BasicAuth(c.Auth)(r)); err != nil {
			panic(err)
		}
	}()
}

type StatsHandler struct {
	balancer *balancer.Balancer
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result := []string{}
	for _, vs := range h.balancer.VServers {
		s := fmt.Sprintf("pool-%s:\n%s", vs.Name, vs.Stats)
		log.Infof(s)
		result = append(result, s)
	}
	result = append(result, "\n")
	io.WriteString(w, strings.Join(result, "\n"))
}

func ListAllVirtualServer(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, vs := range b.VServers {
			data := fmt.Sprintf("Name:%s, Address:%s, Status:%s, Pool:\n%s\n\n",
				vs.Name, vs.Address, vs.Status(), vs.Pool)
			io.WriteString(w, data)
		}
	})
}

func ListVirtualServer(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServ err=%v", err)
			WriteBadRequest(w, err)
			return
		}
		msg := vs.Pool.String()
		io.WriteString(w, msg)
	})
}

type ModifyVirtualServer struct {
	Action string `json:"action"`
}

func ModifyVirtualServerStatus(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		var req ModifyVirtualServer
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			log.Errorf("Decode request err=%v", err)
			WriteBadRequest(w, err)
			return
		}
		action := req.Action
		log.Infof("virtual server name %s, action %s", name, action)
		msg := "success"

		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServ err=%v", err)
			WriteBadRequest(w, err)
			return
		}

		if action == "enable" {
			if err := vs.Run(); err != nil {
				msg = err.Error()
			}
		} else if action == "disable" {
			if err := vs.Stop(); err != nil {
				msg = err.Error()
			}
		} else {
			msg = "unknown action"
		}

		io.WriteString(w, msg)
	})
}

func AddVirtualServer(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var vs config.VirtualServer
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&vs)
		if err != nil {
			log.Errorf("Decode request err=%v", err)
			WriteBadRequest(w, err)
			return
		}

		log.Infof("VirtualServer %v", vs)
		err = b.AddVirtualServer(&vs)
		if err != nil {
			log.Errorf("AddVirtualServ err=%v", err)
			WriteBadRequest(w, err)
			return
		}

		io.WriteString(w, "Add success")
	})
}

func decodeServer(r *http.Request) (*config.Server, error) {
	var server config.Server
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&server)
	if err != nil {
		log.Errorf("Decode request err=%v", err)
		return nil, err
	}
	return &server, nil
}

func AddPoolMember(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServ err=%v", err)
			WriteBadRequest(w, err)
			return
		}

		server, err := decodeServer(r)
		if err != nil {
			WriteBadRequest(w, err)
			return
		}

		weight := server.Weight
		if weight <= 0 {
			weight = 1
		}
		vs.AddPeer(server.Address, weight)
		io.WriteString(w, "Add peer success")
	})
}

func DeletePoolMember(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServ err=%v", err)
			WriteBadRequest(w, err)
			return
		}
		server, err := decodeServer(r)
		if err != nil {
			WriteBadRequest(w, err)
			return
		}

		vs.RemovePeer(server.Address)
		io.WriteString(w, "Remove peer success")
	})
}
