package service

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/config"
	"github.com/onestraw/golb/controller"
	sd "github.com/onestraw/golb/discovery"
)

type Service struct {
	discovery  *sd.ServiceDiscovery
	controller *controller.Controller
	balancer   *balancer.Balancer
}

func New(configFile string) (*Service, error) {
	c, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}

	sdCfg := c.ServiceDiscovery
	dis, err := sd.New(sd.TypeOpt(sdCfg.Type),
		sd.ClusterOpt(sdCfg.Cluster),
		sd.PrefixOpt(sdCfg.Prefix),
		sd.SecurityOpt(sdCfg.CertFile, sdCfg.KeyFile, sdCfg.TrustedCAFile))
	if err != nil {
		log.Warnf("New ServiceDiscovery err=%v", err)
	}

	ctl := controller.New(&c.Controller)
	b, err := balancer.New(c.VServers)
	if err != nil {
		return nil, err
	}

	return &Service{
		discovery:  dis,
		controller: ctl,
		balancer:   b,
	}, nil
}

func (s *Service) Run() error {
	log.Infof("Starting...")
	sigC := make(chan os.Signal)
	signal.Notify(sigC, os.Interrupt, os.Kill, syscall.SIGTERM)

	s.discovery.Run(s.balancer)
	s.controller.Run(s.balancer)
	if err := s.balancer.Run(); err != nil {
		return err
	}

	sig := <-sigC
	log.Infof("Caught signal %v, exiting...", sig)

	return s.balancer.Stop()
}
