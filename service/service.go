package service

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/config"
	"github.com/onestraw/golb/controller"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	Controller *controller.Controller
	Balancer   *balancer.Balancer
}

func New(configFile string) (*Service, error) {
	c := &config.Configuration{}
	err := c.Load(configFile)
	if err != nil {
		return nil, err
	}

	b, err := balancer.New(c.VServers)
	if err != nil {
		return nil, err
	}

	ctl := controller.New(&c.Controller)

	return &Service{
		Balancer:   b,
		Controller: ctl,
	}, nil
}

func (s *Service) Run() error {
	log.Infof("Starting...")
	sigC := make(chan os.Signal)
	signal.Notify(sigC, os.Interrupt, os.Kill, syscall.SIGTERM)

	s.Controller.Run(s.Balancer)
	if err := s.Balancer.Run(); err != nil {
		return err
	}

	sig := <-sigC
	log.Infof("Caught signal %v, exiting...", sig)

	return s.Balancer.Stop()
}
