package discovery

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/discovery/etcd"
)

type ServiceDiscovery struct {
	Enabled       bool
	Type          string
	Cluster       string
	Prefix        string
	CertFile      string
	KeyFile       string
	TrustedCAFile string
}

func New(opts ...ServiceDiscoveryOption) (*ServiceDiscovery, error) {
	sd := &ServiceDiscovery{Enabled: false}
	for _, opt := range opts {
		if err := opt(sd); err != nil {
			return sd, err
		}
	}
	sd.Enabled = true
	return sd, nil
}

type ServiceDiscoveryOption func(*ServiceDiscovery) error

func TypeOpt(t string) ServiceDiscoveryOption {
	return func(sd *ServiceDiscovery) error {
		if t != "etcd" {
			return fmt.Errorf("service discovery type %q currently not supported", t)
		}
		sd.Type = t
		return nil
	}
}

func ClusterOpt(c string) ServiceDiscoveryOption {
	return func(sd *ServiceDiscovery) error {
		if c == "" {
			return fmt.Errorf("Cluster can not be empty")
		}
		sd.Cluster = c
		return nil
	}
}

func PrefixOpt(p string) ServiceDiscoveryOption {
	return func(sd *ServiceDiscovery) error {
		p = strings.TrimSuffix(p, "/")
		if p == "" {
			return fmt.Errorf("Prefix can not be empty")
		}
		if p[0] != '/' {
			return fmt.Errorf("prefix not start with '/'")
		}
		if strings.LastIndex(p, "/") != 0 {
			return fmt.Errorf("prefix contains '/'")
		}
		sd.Prefix = p
		return nil
	}
}
func SecurityOpt(certFile, keyFile, trustedCAFile string) ServiceDiscoveryOption {
	return func(sd *ServiceDiscovery) error {
		if certFile == "" && keyFile == "" {
			log.Infof("Service discovery security (https) is disabled")
			return nil
		}
		if _, err := os.Stat(certFile); err != nil {
			return fmt.Errorf("Cert file '%s' does not exist", certFile)
		}
		if _, err := os.Stat(keyFile); err != nil {
			return fmt.Errorf("Key file '%s' does not exist", keyFile)
		}
		sd.CertFile = certFile
		sd.KeyFile = keyFile
		sd.TrustedCAFile = trustedCAFile
		return nil
	}
}

func (sd *ServiceDiscovery) Run(balancer *balancer.Balancer) {
	if !sd.Enabled {
		log.Infof("ServiceDiscovery is not enabled")
		return
	}

	cli, err := etcd.New(sd.Cluster, sd.Prefix, sd.CertFile, sd.KeyFile, sd.TrustedCAFile)
	if err != nil {
		log.Errorf("etcd.New() err=%v", err)
		return
	}
	go cli.Run(balancer)
}
