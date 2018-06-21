package balancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/config"
)

type Service struct {
	VServers []*VirtualServer
}

func New(configFile string) (*Service, error) {
	c := &config.Configuration{}
	err := c.Load(configFile)
	if err != nil {
		return nil, err
	}

	ss := make([]*VirtualServer, len(c.VServers))
	for i, vs := range c.VServers {
		s := NewVirtualServer(
			AddressOpt(vs.Address),
			ServerNameOpt(vs.ServerName),
			ProtocolOpt(vs.Protocol),
			LBMethodOpt(vs.LBMethod),
			PoolOpt(vs.LBMethod, vs.Pool),
		)
		log.Infof("Listen %s, proto %s, method %s, pool %v", s.Address, s.Protocol, s.LBMethod, s.Pool)
		ss[i] = s
	}

	return &Service{VServers: ss}, nil
}

func (s *Service) Run() error {
	log.Infof("Starting...")

	sigC := make(chan os.Signal)
	signal.Notify(sigC, os.Interrupt, os.Kill, syscall.SIGTERM)

	for _, vs := range s.VServers {
		go func(s *VirtualServer) {
			if s.Protocol == PROTO_HTTP {
				http.ListenAndServe(s.Address, Redirect(s))
			} else {
				panic(ErrNotSupportedProto)
			}
		}(vs)
	}

	sig := <-sigC
	log.Infof("Caught signal %v, exiting...", sig)
	return nil
}

// Redirect dispatch the request between backend servers
// TODO:
// 	1. buffer the request, check the response code
//  2. disable peer if code is 5xx, then retry with another peer
func Redirect(s *VirtualServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.ServerName {
			log.Errorf("Host not match, host=%s", r.Host)
			WriteError(w, ErrHostNotMatch)
			return
		}
		// use client's address as hash key if using consistent-hash method
		peer := s.Pool.Get(r.RemoteAddr)
		if peer == "" {
			log.Errorf("Peer not found")
			WriteError(w, ErrPeerNotFound)
			return
		}

		rp, ok := s.ReverseProxy[peer]
		if !ok {
			target, err := url.Parse("http://" + peer)
			if err != nil {
				log.Errorf("url.Parse peer=%s, error=%v", peer, err)
				WriteError(w, ErrInternalBalancer)
				return
			}
			log.Infof("%v", target)
			rp = httputil.NewSingleHostReverseProxy(target)
			s.ReverseProxy[peer] = rp
		}
		rp.ServeHTTP(w, r)
	}
}
