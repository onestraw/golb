package balancer

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/config"
)

type Balancer struct {
	sync.RWMutex
	VServers []*VirtualServer
}

func New(vss []config.VirtualServer) (*Balancer, error) {
	b := &Balancer{
		VServers: []*VirtualServer{},
	}
	for _, vs := range vss {
		if err := b.AddVirtualServer(&vs); err != nil {
			return b, err
		}
	}

	return b, nil
}

func (b *Balancer) AddVirtualServer(vs *config.VirtualServer) error {
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

	b.Lock()
	defer b.Unlock()
	for _, v := range b.VServers {
		if v.Name == new_vs.Name {
			return ErrVirtualServerNameExisted
		}
		if v.Address == new_vs.Address {
			return ErrVirtualServerAddressExisted
		}
	}
	b.VServers = append(b.VServers, new_vs)

	return nil
}

func (b *Balancer) FindVirtualServer(name string) (*VirtualServer, error) {
	b.RLock()
	defer b.RUnlock()
	for _, v := range b.VServers {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, ErrVirtualServerNotFound
}

func (b *Balancer) Run() error {
	for _, vs := range b.VServers {
		if err := vs.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (b *Balancer) Stop() error {
	for _, vs := range b.VServers {
		if err := vs.Stop(); err != nil {
			return err
		}
	}
	return nil
}
