package config

import (
	"encoding/json"
	"errors"
	"os"
)

var (
	ErrVirtualServerDuplicated   = errors.New("Vritual Server Duplicated")
	ErrPoolMemberDuplicated      = errors.New("Pool Member Duplicated")
	ErrVirtualServerNameEmpty    = errors.New("Vritual Server Name is not specified")
	ErrVirtualServerAddressEmpty = errors.New("Vritual Server Address is not specified")
)

type Server struct {
	Address string `json:"address"`
	Weight  int    `json:"weight"`
}

type VirtualServer struct {
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	ServerName string   `json:"server_name"`
	Protocol   string   `json:"protocol"`
	LBMethod   string   `json:"lb_method"`
	Pool       []Server `json:"pool"`
}

type Authentication struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Controller struct {
	Address string         `json:"address"`
	Auth    Authentication `json:"auth"`
}

type Configuration struct {
	Controller Controller      `json:"controller"`
	VServers   []VirtualServer `json:"virtual_server"`
}

func (c *Configuration) Load(configFile string) error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(c); err != nil {
		return err
	}
	if err = c.check(); err != nil {
		return err
	}
	return nil
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
		} else {
			set[vs.Name] = true
		}

		if len(vs.Pool) > 1 {
			pset := make(map[string]bool)
			for _, p := range vs.Pool {
				if _, ok := pset[p.Address]; ok {
					return ErrPoolMemberDuplicated
				} else {
					pset[p.Address] = true
				}
			}
		}
	}
	return nil
}
