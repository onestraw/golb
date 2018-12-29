package main

import (
	"flag"
	"fmt"

	"github.com/hashicorp/consul/api"
)

// ignore all errors
func main() {
	var consulAddr, name, tag string
	flag.StringVar(&consulAddr, "consul-addr", "127.0.0.1:8500", "consul address")
	flag.StringVar(&name, "name", "web", "name of the service")
	flag.StringVar(&tag, "tag", "", "")
	flag.Parse()

	consulClient, _ := NewConsulClient(consulAddr)
	instances, meta, _ := consulClient.Service(name, tag)
	fmt.Println("List all instances through consul:", meta)

	for _, inst := range instances {
		fmt.Println(inst.ServiceAddress)
		fmt.Println(inst.ServiceMeta)
	}
}

//Client provides an interface for getting data out of Consul
type Client interface {
	// Get a Service from consul
	Service(string, string) ([]*api.CatalogService, *api.QueryMeta, error)
}

type client struct {
	consul *api.Client
}

//NewConsul returns a Client interface for given consul address
func NewConsulClient(addr string) (Client, error) {
	config := api.DefaultConfig()
	config.Address = addr
	c, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &client{consul: c}, nil
}

// Service return a service
func (c *client) Service(service, tag string) ([]*api.CatalogService, *api.QueryMeta, error) {
	ss, meta, err := c.consul.Catalog().Service(service, tag, nil)
	if len(ss) == 0 && err == nil {
		return nil, nil, fmt.Errorf("service ( %s ) was not found", service)
	}
	if err != nil {
		return nil, nil, err
	}
	return ss, meta, nil
}
