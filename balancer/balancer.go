package balancer

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/config"
)

type Service struct {
	Controller *Controller
	VServers   []*VirtualServer
}

func New(configFile string) (*Service, error) {
	c := &config.Configuration{}
	err := c.Load(configFile)
	if err != nil {
		return nil, err
	}

	ss := make([]*VirtualServer, len(c.VServers))
	for i, vs := range c.VServers {
		s, err := NewVirtualServer(
			NameOpt(vs.Name),
			AddressOpt(vs.Address),
			ServerNameOpt(vs.ServerName),
			ProtocolOpt(vs.Protocol),
			LBMethodOpt(vs.LBMethod),
			PoolOpt(vs.LBMethod, vs.Pool),
		)
		if err != nil {
			return nil, err
		}
		log.Infof("Listen %s, proto %s, method %s, pool %v", s.Address, s.Protocol, s.LBMethod, s.Pool)
		ss[i] = s
	}

	ctlCfg := c.Controller
	ctl := &Controller{
		Address: ctlCfg.Address,
		Auth:    &Authentication{ctlCfg.Auth.Username, ctlCfg.Auth.Password},
	}

	return &Service{VServers: ss, Controller: ctl}, nil
}

func (s *Service) Run() error {
	log.Infof("Starting...")
	sigC := make(chan os.Signal)
	signal.Notify(sigC, os.Interrupt, os.Kill, syscall.SIGTERM)

	go s.Controller.Run(s)

	for _, vs := range s.VServers {
		go vs.Run()
	}

	sig := <-sigC
	log.Infof("Caught signal %v, exiting...", sig)
	return nil
}
