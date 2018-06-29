package balancer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

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
	if err != nil {
		t.Errorf("Load err= %v", err)
		return nil
	}

	b, err := New(c.VServers)
	if err != nil {
		t.Errorf("New() err= %v", err)
		return nil
	}
	return b
}

type Response struct {
	StatusCode int
	Body       string
}

func request() (*Response, error) {
	client := &http.Client{}
	proxyUrl := fmt.Sprintf("http://%s/", proxyAddr)
	req, err := http.NewRequest("GET", proxyUrl, nil)
	if err != nil {
		return nil, err
	}
	//req.Header.Set("Host", "localhost")
	req.Host = "localhost"
	fmt.Printf("%v", req.Header)
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

func TestBalancer(t *testing.T) {
	b := mockBalancer(t)
	if err := b.Run(); err != nil {
		t.Errorf("run balancer err=%v", err)
		return
	}
	time.Sleep(2 * time.Second)
	//because goroutine in vs.Run() maybe unfinished, vs.status is unpredictable
	//t.Logf("balancer.VServers[0]: %v", b.VServers[0])

	result := map[string]int{}
	for i := 0; i < 10; i += 1 {
		resp, err := request()
		if err != nil {
			t.Errorf("http.Get() err=%v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%dth expect status %d, but got %d", i, http.StatusOK, resp.StatusCode)
			return
		}

		result[resp.Body] += 1
	}

	if result["s1"] != 5 || result["s2"] != 5 {
		t.Errorf("LB stats should be (5,5), but got %v,", result)
		return
	}

	if err := b.Stop(); err != nil {
		t.Errorf("balancer.Stop() err=%v", err)
	}
}
