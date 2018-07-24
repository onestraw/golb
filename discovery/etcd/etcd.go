// package etcd watch the configuration stored in etcd
// data format
// prefix/virtualserver
//		- /name
//			- /address
//			- /pool
//				- /address
//					- /address
//					- /weight
//				- /address
//					- /weight
//
// currently we only support add/remove peer in virtualserver
// (/<prefix>/virtualserver/<vs_name>/pool/<127.0.0.1:8001>/address, 127.0.0.1:8001)

package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/coreos/etcd/pkg/transport"
	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/balancer"
)

// Client wraps a etcd client.
type Client struct {
	prefix string
	cli    *clientv3.Client
}

// New returns a Client object.
func New(endpoints, prefix, certFile, keyFile, trustedCAFile string) (*Client, error) {
	var err error
	var tlsConfig *tls.Config
	if certFile != "" && keyFile != "" {
		tlsInfo := transport.TLSInfo{
			CertFile:      certFile,
			KeyFile:       keyFile,
			TrustedCAFile: trustedCAFile,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(endpoints, ","),
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		prefix: prefix,
		cli:    cli,
	}, nil
}

type session struct {
	// event has only two type: add or delete
	isDelete bool
	rawKey   string
	rawValue string
	vsName   string
	peer     string
}

// key label
const (
	VSPrefix     = "virtualserver"
	PoolPrefix   = "pool"
	AddressLabel = "address"
)

func newSession(ev *clientv3.Event) (*session, error) {
	s := &session{}
	if ev.Type == mvccpb.DELETE {
		s.isDelete = true
	}
	s.rawKey = string(ev.Kv.Key)
	s.rawValue = string(ev.Kv.Value)

	keys := strings.Split(s.rawKey, "/")
	if len(keys) != 7 || keys[2] != VSPrefix || keys[4] != PoolPrefix || keys[6] != AddressLabel {
		return nil, fmt.Errorf("unidentified key: %q", s.rawKey)
	}

	s.vsName = keys[3]
	s.peer = keys[5]
	if !s.isDelete && s.peer != s.rawValue {
		return nil, fmt.Errorf("the value should be the same as the peer tag of key")
	}
	return s, nil
}

// Run traps into a dead loop to watch and update key changes.
func (c *Client) Run(balancer *balancer.Balancer) {
	log.Infof(`Currently we only support add/remove peer in virtualserver, the key format:
	/<prefix>/virtualserver/<virtualserver_name>/pool/<peer_address>/address`)

	// a. read the existing key
	resp, err := c.cli.Get(context.Background(), c.prefix, clientv3.WithPrefix())
	if err != nil {
		log.Errorf("Get %q err=%v", c.prefix, err)
	} else {
		for _, kv := range resp.Kvs {
			ev := &clientv3.Event{Kv: kv, Type: mvccpb.PUT}
			if err := c.dispatch(balancer, ev); err != nil {
				log.Errorf("handle '%v' err=%v", ev, err)
			}
		}
	}
	// b. watch the updates
	defer c.cli.Close()
	for {
		rch := c.cli.Watch(context.Background(), c.prefix, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				if err := c.dispatch(balancer, ev); err != nil {
					log.Errorf("handle '%v' err=%v", ev, err)
				}
			}
		}
	}
}

func (c *Client) dispatch(balancer *balancer.Balancer, ev *clientv3.Event) error {
	log.Infof("%s %q : %q", ev.Type, ev.Kv.Key, ev.Kv.Value)
	s, err := newSession(ev)
	if err != nil {
		return err
	}
	vs, err := balancer.FindVirtualServer(s.vsName)
	if err != nil {
		return err
	}

	if s.isDelete {
		vs.RemovePeer(s.peer)
	} else {
		vs.AddPeer(s.peer)
	}
	return nil
}
