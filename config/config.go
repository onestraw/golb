package config

import (
	"encoding/json"
	"os"
)

type Server struct {
	Address string `json:"address"`
	Weight  int    `json:"weight"`
}

type VirtualServer struct {
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
	return decoder.Decode(c)
}
