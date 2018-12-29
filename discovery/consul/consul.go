package consul

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"

	"github.com/onestraw/golb/balancer"
)

const (
	queryInterval = time.Second * 5
)

// Client wraps a consul client.
type Client struct {
	consul *api.Client

	// service: {server: weight}
	virtualServers map[string]map[string]string
}

//New returns a Client interface for given consul address.
func New(addr string) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = addr
	c, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Client{
		consul:         c,
		virtualServers: make(map[string]map[string]string),
	}, nil
}

// Service return a service.
func (c *Client) Service(service, tag string) ([]*api.CatalogService, *api.QueryMeta, error) {
	ss, meta, err := c.consul.Catalog().Service(service, tag, nil)
	if len(ss) == 0 && err == nil {
		return nil, nil, fmt.Errorf("service ( %s ) was not found", service)
	}
	if err != nil {
		return nil, nil, err
	}
	return ss, meta, nil
}

// Run query services periodically.
func (c *Client) Run(balancer *balancer.Balancer) {
	ticker := time.NewTicker(queryInterval)
	for range ticker.C {
		c.run(balancer)
	}
}

func (c *Client) run(balancer *balancer.Balancer) {
	for _, vs := range balancer.VServers {
		// vs.Name is used as service name.
		// tag is "".
		instances, _, err := c.Service(vs.Name, "")
		if err != nil {
			log.Errorf("consul: get service %q err=%v", vs.Name, err)
			continue
		}

		if _, ok := c.virtualServers[vs.Name]; !ok {
			c.virtualServers[vs.Name] = map[string]string{}
		}

		serverSet := map[string]struct{}{}
		for _, inst := range instances {
			server := inst.ServiceAddress
			serverSet[server] = struct{}{}
			val, ok := inst.ServiceMeta["weight"]
			if !ok {
				val = "1"
			}
			w, ok := c.virtualServers[vs.Name][server]
			if !ok || val != w {
				wi, err := strconv.Atoi(val)
				if err != nil {
					log.Errorf("consul: strconv.Atoi(%q) err=%v", val, err)
					wi = 1
				}
				vs.AddPeer(server, wi)
			}
			c.virtualServers[vs.Name][server] = val
		}

		for s := range c.virtualServers[vs.Name] {
			if _, ok := serverSet[s]; !ok {
				vs.RemovePeer(s)
			}
		}
	}
}
