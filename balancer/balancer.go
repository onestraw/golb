package balancer

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/config"
)

type Service struct {
	sync.RWMutex
	Controller *Controller
	VServers   []*VirtualServer
}

func New(configFile string) (*Service, error) {
	c := &config.Configuration{}
	err := c.Load(configFile)
	if err != nil {
		return nil, err
	}

	ctlCfg := c.Controller
	ctl := &Controller{
		Address: ctlCfg.Address,
		Auth:    &Authentication{ctlCfg.Auth.Username, ctlCfg.Auth.Password},
	}
	s := &Service{
		VServers:   []*VirtualServer{},
		Controller: ctl,
	}
	for _, vs := range c.VServers {
		if err := s.AddVirtualServer(&vs); err != nil {
			return s, err
		}
	}

	return s, nil
}

func (s *Service) AddVirtualServer(vs *config.VirtualServer) error {
	new_vs, err := NewVirtualServer(
		NameOpt(vs.Name),
		AddressOpt(vs.Address),
		ServerNameOpt(vs.ServerName),
		ProtocolOpt(vs.Protocol),
		LBMethodOpt(vs.LBMethod),
		PoolOpt(vs.LBMethod, vs.Pool),
	)
	if err != nil {
		return err
	}
	log.Infof("Listen %s, proto %s, method %s, pool %v",
		new_vs.Address, new_vs.Protocol, new_vs.LBMethod, new_vs.Pool)

	s.Lock()
	defer s.Unlock()
	for _, v := range s.VServers {
		if v.Name == new_vs.Name {
			return ErrVirtualServerNameExisted
		}
		if v.Address == new_vs.Address {
			return ErrVirtualServerAddressExisted
		}
	}
	s.VServers = append(s.VServers, new_vs)

	return nil
}

func (s *Service) FindVirtualServer(name string) (*VirtualServer, error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.VServers {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, ErrVirtualServerNotFound
}

func (s *Service) Run() error {
	log.Infof("Starting...")
	sigC := make(chan os.Signal)
	signal.Notify(sigC, os.Interrupt, os.Kill, syscall.SIGTERM)

	go s.Controller.Run(s)

	for _, vs := range s.VServers {
		vs.Run()
	}

	sig := <-sigC
	log.Infof("Caught signal %v, exiting...", sig)
	return nil
}
