package config

import (
	"io/ioutil"
	"syscall"
	"testing"
)

func TestLoad(t *testing.T) {
	f, err := ioutil.TempFile("", "testconf.json")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(f.Name())

	jsonBody := `{"virtual_server":[{"address":"127.0.0.1:8081","server_name":"localhost","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10002","weight":2}],"lb_method":"round-robin"}]}`
	ioutil.WriteFile(f.Name(), []byte(jsonBody), 0644)

	c := &Configuration{}
	err = c.Load(f.Name())
	if err != nil {
		t.Errorf("Load error: %v", err)
	}
	if len(c.VServers) != 1 {
		t.Errorf("The number of virtual_server should be 1")
	}

	vs := c.VServers[0]
	if vs.Address != "127.0.0.1:8081" || vs.LBMethod != "round-robin" || vs.Protocol != "" || vs.ServerName != "localhost" || len(vs.Pool) != 2 {
		t.Errorf("Load configuration error, got %v", c)
	}

	s := vs.Pool[1]
	if s.Address != "127.0.0.1:10002" || s.Weight != 2 {
		t.Errorf("Parse server error, got %v", s)
	}
}
