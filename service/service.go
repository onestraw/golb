package service

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/config"
	"github.com/onestraw/golb/controller"
)

type Service struct {
	controller *controller.Controller
	balancer   *balancer.Balancer
}

func New(configFile string) (*Service, error) {
	c := &config.Configuration{}
	if err := c.Load(configFile); err != nil {
		return nil, err
	}

	ctl := controller.New(&c.Controller)
	b, err := balancer.New(c.VServers)
	if err != nil {
		return nil, err
	}

	return &Service{
		controller: ctl,
		balancer:   b,
	}, nil
}

func (s *Service) Run() error {
	log.Infof("Starting...")
	sigC := make(chan os.Signal)
	signal.Notify(sigC, os.Interrupt, os.Kill, syscall.SIGTERM)

	s.controller.Run(s.balancer)
	if err := s.balancer.Run(); err != nil {
		return err
	}

	sig := <-sigC
	log.Infof("Caught signal %v, exiting...", sig)

	return s.balancer.Stop()
}
