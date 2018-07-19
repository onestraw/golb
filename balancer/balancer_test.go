package balancer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onestraw/golb/config"
)

const (
	proxyAddr = "127.0.0.1:8081"
)

func load(jsonBody string) (*config.Configuration, error) {
	f, err := ioutil.TempFile("", "testconf.json")
	if err != nil {
		return nil, err
	}
	defer syscall.Unlink(f.Name())

	ioutil.WriteFile(f.Name(), []byte(jsonBody), 0644)

	c := &config.Configuration{}
	err = c.Load(f.Name())
	if err != nil {
		return nil, err
	}
	return c, nil
}

type Response struct {
	StatusCode int
	Body       string
}

func request(addr string) (*Response, error) {
	client := &http.Client{}
	proxyUrl := fmt.Sprintf("http://%s/", addr)
	req, err := http.NewRequest("GET", proxyUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Host = "localhost"
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}, nil
}

func newHandler(label string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(label))
	})
}

func mockBalancer(t *testing.T) *Balancer {
	s1 := httptest.NewServer(newHandler("s1"))
	s2 := httptest.NewServer(newHandler("s2"))
	jsonBody := fmt.Sprintf(`{"virtual_server":[{"name":"web","address":"%s","pool":[{"address":"%s","weight":1},{"address":"%s","weight":1}],"lb_method":"round-robin"}]}`, proxyAddr, s1.URL[7:], s2.URL[7:])

	c, err := load(jsonBody)
	require.NoError(t, err)

	b, err := New(c.VServers)
	require.NoError(t, err)

	return b
}

func TestBalancer(t *testing.T) {
	b := mockBalancer(t)
	require.NoError(t, b.Run())
	time.Sleep(2 * time.Second)
	//because goroutine in vs.Run() maybe unfinished, vs.status is unpredictable
	//t.Logf("balancer.VServers[0]: %v", b.VServers[0])
	result := map[string]int{}
	for i := 0; i < 10; i += 1 {
		resp, err := request(proxyAddr)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		result[resp.Body] += 1
	}
	assert.Equal(t, 5, result["s1"])
	assert.Equal(t, 5, result["s2"])

	require.NoError(t, b.Stop())
}

func TestFindVirtualServer(t *testing.T) {
	b := mockBalancer(t)
	vsName := "web"
	vs, err := b.FindVirtualServer(vsName)
	require.NoError(t, err)
	assert.NotNil(t, vs)

	vs, err = b.FindVirtualServer("not_existed")
	assert.Equal(t, ErrVirtualServerNotFound, err)
	assert.Nil(t, vs)
}
