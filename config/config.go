package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Configuration error.
var (
	ErrVirtualServerDuplicated   = errors.New("vritual server duplicated")
	ErrPoolMemberDuplicated      = errors.New("pool member duplicated")
	ErrVirtualServerNameEmpty    = errors.New("vritual server name is not specified")
	ErrVirtualServerAddressEmpty = errors.New("vritual server address is not specified")
)

// Server configuration.
type Server struct {
	Address string `json:"address" yaml:"address"`
	Weight  int    `json:"weight" yaml:"weight"`
}

// VirtualServer configuration.
type VirtualServer struct {
	Name       string   `json:"name" yaml:"name"`
	Address    string   `json:"address" yaml:"address"`
	ServerName string   `json:"server_name" yaml:"server_name"`
	Protocol   string   `json:"protocol" yaml:"protocol"`
	CertFile   string   `json:"cert_file" yaml:"cert_file"`
	KeyFile    string   `json:"key_file" yaml:"key_file"`
	LBMethod   string   `json:"lb_method" yaml:"lb_method"`
	Pool       []Server `json:"pool" yaml:"pool"`
}

// Authentication configuration.
type Authentication struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// Controller configuration.
type Controller struct {
	Address string         `json:"address" yaml:"address"`
	Auth    Authentication `json:"auth" yaml:"auth"`
}

// ServiceDiscovery configuration.
type ServiceDiscovery struct {
	Type          string `json:"type" yaml:"type"`
	Cluster       string `json:"cluster" yaml:"cluster"`
	Prefix        string `json:"prefix" yaml:"prefix"`
	CertFile      string `json:"cert_file" yaml:"cert_file"`
	KeyFile       string `json:"key_file" yaml:"key_file"`
	TrustedCAFile string `json:"trusted_ca_file" yaml:"trusted_ca_file"`
}

// Configuration is the whole json configuration.
type Configuration struct {
	ServiceDiscovery ServiceDiscovery `json:"service_discovery" yaml:"service_discovery"`
	Controller       Controller       `json:"controller" yaml:"controller"`
	VServers         []VirtualServer  `json:"virtual_server" yaml:"virtual_server"`
}

func loadJSON(configFile string) (*Configuration, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	c := &Configuration{}
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}

func loadYAML(configFile string) (*Configuration, error) {
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var c Configuration
	if err = yaml.Unmarshal(contents, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Load reads the configFile and returns a Configuration object.
func Load(configFile string) (*Configuration, error) {
	var err error
	var c *Configuration

	if strings.HasSuffix(configFile, "json") {
		c, err = loadJSON(configFile)
	} else {
		c, err = loadYAML(configFile)
	}
	if err != nil {
		return nil, err
	}

	if err = c.check(); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadFromString returns a Configuration object.
func LoadFromString(config string) (*Configuration, error) {
	var err error
	c := &Configuration{}
	decoder := json.NewDecoder(strings.NewReader(config))
	if err = decoder.Decode(c); err != nil {
		return nil, err
	}
	if err = c.check(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Configuration) check() error {
	set := make(map[string]bool)
	for _, vs := range c.VServers {
		if vs.Name == "" {
			return ErrVirtualServerNameEmpty
		}

		if vs.Address == "" {
			return ErrVirtualServerAddressEmpty
		}

		if _, ok := set[vs.Name]; ok {
			return ErrVirtualServerDuplicated
		}
		set[vs.Name] = true

		if len(vs.Pool) > 1 {
			pset := make(map[string]bool)
			for _, p := range vs.Pool {
				if _, ok := pset[p.Address]; ok {
					return ErrPoolMemberDuplicated
				}
				pset[p.Address] = true
			}
		}
	}
	return nil
}
