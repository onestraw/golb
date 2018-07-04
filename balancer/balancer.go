package balancer

import (
	"sync"

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

func (b *Balancer) AddVirtualServer(cvs *config.VirtualServer) error {
	vs, err := NewVirtualServer(
		NameOpt(cvs.Name),
		AddressOpt(cvs.Address),
		ServerNameOpt(cvs.ServerName),
		ProtocolOpt(cvs.Protocol),
		TLSOpt(cvs.CertFile, cvs.KeyFile),
		LBMethodOpt(cvs.LBMethod),
		PoolOpt(cvs.LBMethod, cvs.Pool),
	)
	if err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()
	for _, v := range b.VServers {
		if v.Name == vs.Name {
			return ErrVirtualServerNameExisted
		}
		if v.Address == vs.Address {
			return ErrVirtualServerAddressExisted
		}
	}
	b.VServers = append(b.VServers, vs)

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
