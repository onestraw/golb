package balancer

import (
	"sync"

	"github.com/onestraw/golb/config"
)

// Balancer is a set of VirtualServers.
type Balancer struct {
	sync.RWMutex
	VServers []*VirtualServer
}

// New returns a Balancer object.
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

// AddVirtualServer loads from config.VirtualServer.
func (b *Balancer) AddVirtualServer(cvs *config.VirtualServer) error {
	vs, err := NewVirtualServer(
		NameOpt(cvs.Name),
		AddressOpt(cvs.Address),
		ServerNameOpt(cvs.ServerName),
		ProtocolOpt(cvs.Protocol),
		TLSOpt(cvs.CertFile, cvs.KeyFile),
		LBMethodOpt(cvs.LBMethod),
		PoolOpt(cvs.Pool),
		RetryOpt(true),
	)
	if err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()
	b.VServers = append(b.VServers, vs)

	return nil
}

// FindVirtualServer search b.VServers by name.
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

// Run starts all VirtualServers.
func (b *Balancer) Run() error {
	for _, vs := range b.VServers {
		if err := vs.Run(); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops all VirtualServers.
func (b *Balancer) Stop() error {
	for _, vs := range b.VServers {
		if err := vs.Stop(); err != nil {
			return err
		}
	}
	return nil
}
