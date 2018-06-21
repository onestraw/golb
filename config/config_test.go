package config

import (
	"io/ioutil"
	"syscall"
	"testing"
)

func load(jsonBody string) (*Configuration, error) {
	f, err := ioutil.TempFile("", "testconf.json")
	if err != nil {
		return nil, err
	}
	defer syscall.Unlink(f.Name())

	ioutil.WriteFile(f.Name(), []byte(jsonBody), 0644)

	c := &Configuration{}
	err = c.Load(f.Name())
	if err != nil {
		return nil, err
	}
	return c, nil
}

func TestLoad(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081","server_name":"localhost","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10002","weight":2}],"lb_method":"round-robin"}]}`

	c, err := load(jsonBody)
	if err != nil {
		t.Errorf("Load error: %v", err)
		return
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

func TestLoadEmpty(t *testing.T) {
	c, err := load("{}")
	if err != nil {
		t.Errorf("Load error: %v", err)
		return
	}
	t.Logf("%v", c)
}

func TestCheckVirtualServerDuplicated(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081"},{"name":"web","address":"127.0.0.1:8082"}]}`
	_, err := load(jsonBody)
	if err != ErrVirtualServerDuplicated {
		t.Errorf("Load error: %v, expect: %v", err, ErrVirtualServerDuplicated)
	}
}

func TestCheckPoolMemberDuplicated(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10001","weight":2}]}]}`
	_, err := load(jsonBody)
	if err != ErrPoolMemberDuplicated {
		t.Errorf("Load error: %v, expect: %v", err, ErrPoolMemberDuplicated)
	}
}

func TestCheckVirtualServerName(t *testing.T) {
	jsonBody := `{"virtual_server":[{"address":"127.0.0.1:8081"}]}`
	_, err := load(jsonBody)
	if err != ErrVirtualServerNameEmpty {
		t.Errorf("Load error: %v, expect: %v", err, ErrVirtualServerNameEmpty)
	}
}

func TestCheckVirtualServerAddress(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web"}]}`
	_, err := load(jsonBody)
	if err != ErrVirtualServerAddressEmpty {
		t.Errorf("Load error: %v, expect: %v", err, ErrVirtualServerAddressEmpty)
	}
}
