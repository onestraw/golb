// Package controller provides REST API to configure balancer
//
// controller API
//
// - Authentication
// 	Basic HTTP Auth
//
// - Stats
//	GET http://{controller_address}/stats
//
// - List All LB instance
//	GET http://{controller_address}/vs
//
// - Add LB instance
//	POST http://{controller_address}/vs
//	Body {"name":"redis","address":"127.0.0.1:6379"}
//	Example: curl -XPOST -u admin:admin -H 'content-type: application/json' -d '{"name":"redis","address":"127.0.0.1:6379"}' http://127.0.0.1:6587/vs
//
// - Enable LB instance
//	POST http://{controller_address}/vs/{name}
//	Body {"action":"enable"}
//
// - Disable LB instance
//	POST http://{controller_address}/vs/{name}
//	Body {"action":"disable"}
//
// - List pool member of LB instance
//	GET http://{controller_address}/vs/{name}
//
// - Add pool member to LB instance
//	POST http://{controller_address}/vs/{name}/pool
//	Body: {"address":"127.0.0.1:10003","weight":2}
//	Example: curl -XPOST -u admin:admin -H 'content-type: application/json' -d '{"address":"127.0.0.1:10003"}' http://127.0.0.1:6587/vs/web/pool
//
// - Remove pool member from LB instance
//	DELETE http://{controller_address}/vs/{name}/pool
//	Body: {"address":"127.0.0.1:10002"}
//	Example: curl -XDELETE -u admin:admin -H 'content-type: application/json' -d '{"address":"127.0.0.1:10002"}' http://127.0.0.1:6587/vs/web/pool
//
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

// Controller provides interface to operate balancer.
type Controller struct {
	Address string
	Auth    *Authentication
}

// New returns a Controller object.
func New(ctlCfg *config.Controller) *Controller {
	return &Controller{
		Address: ctlCfg.Address,
		Auth:    &Authentication{ctlCfg.Auth.Username, ctlCfg.Auth.Password},
	}
}

// Run starts the controller.
func (c *Controller) Run(balancer *balancer.Balancer) {
	r := mux.NewRouter()
	r.Handle("/stats", statsHandler(balancer)).Methods("GET")
	r.Handle("/vs", addVirtualServer(balancer)).Methods("POST")
	r.Handle("/vs", listAllVirtualServer(balancer)).Methods("GET")
	r.Handle("/vs/{name}", modifyVirtualServerStatus(balancer)).Methods("POST")
	r.Handle("/vs/{name}", listVirtualServer(balancer)).Methods("GET")
	r.Handle("/vs/{name}/pool", addPoolMember(balancer)).Methods("POST")
	r.Handle("/vs/{name}/pool", deletePoolMember(balancer)).Methods("DELETE")
	go func() {
		if err := http.ListenAndServe(c.Address, BasicAuth(c.Auth)(r)); err != nil {
			panic(err)
		}
	}()
}

func statsHandler(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := []string{}
		for _, vs := range b.VServers {
			s := vs.Stats()
			log.Infof(s)
			result = append(result, s)
		}
		io.WriteString(w, strings.Join(result, "\n"))
	})
}

func listAllVirtualServer(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, vs := range b.VServers {
			data := fmt.Sprintf("Name:%s, Address:%s, Status:%s, Pool:\n%s\n\n",
				vs.Name, vs.Address, vs.Status(), vs.Pool)
			io.WriteString(w, data)
		}
	})
}

func listVirtualServer(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServer err=%v", err)
			WriteBadRequest(w, err)
			return
		}
		msg := vs.Pool.String()
		io.WriteString(w, msg)
	})
}

type modifyVirtualServer struct {
	Action string `json:"action"`
}

func modifyVirtualServerStatus(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		var req modifyVirtualServer
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
			log.Errorf("FindVirtualServer err=%v", err)
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
			log.Errorf("%v", ErrUnknownAction)
			WriteError(w, ErrUnknownAction)
			return
		}

		io.WriteString(w, msg)
	})
}

func addVirtualServer(b *balancer.Balancer) http.Handler {
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
			log.Errorf("AddVirtualServer err=%v", err)
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

func addPoolMember(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServer err=%v", err)
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

func deletePoolMember(b *balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		vs, err := b.FindVirtualServer(name)
		if err != nil {
			log.Errorf("FindVirtualServer err=%v", err)
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
