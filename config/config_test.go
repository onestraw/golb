package config

import (
	"io/ioutil"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081","server_name":"localhost","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10002","weight":2}],"lb_method":"round-robin"}]}`
	f, err := ioutil.TempFile("", "testconf.json")
	require.NoError(t, err)
	defer syscall.Unlink(f.Name())

	ioutil.WriteFile(f.Name(), []byte(jsonBody), 0644)
	c, err := Load(f.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(c.VServers))

	vs := c.VServers[0]
	assert.Equal(t, "127.0.0.1:8081", vs.Address)
	assert.Equal(t, "round-robin", vs.LBMethod)
	assert.Equal(t, "", vs.Protocol)
	assert.Equal(t, "localhost", vs.ServerName)
	assert.Equal(t, 2, len(vs.Pool))

	s := vs.Pool[1]
	assert.Equal(t, "127.0.0.1:10002", s.Address)
	assert.Equal(t, 2, s.Weight)
}

func TestLoadNotJsonFile(t *testing.T) {
	f, err := ioutil.TempFile("", "testconf.json")
	require.NoError(t, err)
	defer syscall.Unlink(f.Name())

	ioutil.WriteFile(f.Name(), []byte("not json"), 0644)
	c, err := Load(f.Name())
	assert.Nil(t, c)
	assert.NotNil(t, err)
}

func TestLoadNotExisted(t *testing.T) {
	c, err := Load("no_file.json")
	assert.Nil(t, c)
	assert.NotNil(t, err)
}

func TestLoadFromString(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081","server_name":"localhost","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10002","weight":2}],"lb_method":"round-robin"}]}`

	c, err := LoadFromString(jsonBody)
	require.NoError(t, err)
	assert.Equal(t, 1, len(c.VServers))

	vs := c.VServers[0]
	assert.Equal(t, "127.0.0.1:8081", vs.Address)
	assert.Equal(t, "round-robin", vs.LBMethod)
	assert.Equal(t, "", vs.Protocol)
	assert.Equal(t, "localhost", vs.ServerName)
	assert.Equal(t, 2, len(vs.Pool))

	s := vs.Pool[1]
	assert.Equal(t, "127.0.0.1:10002", s.Address)
	assert.Equal(t, 2, s.Weight)
}

func TestLoadEmpty(t *testing.T) {
	c, err := LoadFromString("{}")
	require.NoError(t, err)
	t.Logf("%v", c)
}

func TestLoadNotJsonString(t *testing.T) {
	c, err := LoadFromString("error")
	assert.Nil(t, c)
	assert.NotNil(t, err)
}

func TestCheckVirtualServerDuplicated(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081"},{"name":"web","address":"127.0.0.1:8082"}]}`
	c, err := LoadFromString(jsonBody)
	assert.Equal(t, ErrVirtualServerDuplicated, err)
	assert.Nil(t, c)
}

func TestCheckPoolMemberDuplicated(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8081","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10001","weight":2}]}]}`
	c, err := LoadFromString(jsonBody)
	assert.Equal(t, ErrPoolMemberDuplicated, err)
	assert.Nil(t, c)
}

func TestCheckVirtualServerName(t *testing.T) {
	jsonBody := `{"virtual_server":[{"address":"127.0.0.1:8081"}]}`
	c, err := LoadFromString(jsonBody)
	assert.Equal(t, ErrVirtualServerNameEmpty, err)
	assert.Nil(t, c)
}

func TestCheckVirtualServerAddress(t *testing.T) {
	jsonBody := `{"virtual_server":[{"name":"web"}]}`
	c, err := LoadFromString(jsonBody)
	assert.Equal(t, ErrVirtualServerAddressEmpty, err)
	assert.Nil(t, c)
}
