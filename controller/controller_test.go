package controller

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/config"
	"github.com/onestraw/golb/stats"
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

func mockBalancer(t *testing.T) *balancer.Balancer {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8082","server_name":"localhost","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10002","weight":2}],"lb_method":"round-robin"}]}`
	c, err := load(jsonBody)
	if err != nil {
		t.Errorf("Load err= %v", err)
		return nil
	}

	b, err := balancer.New(c.VServers)
	if err != nil {
		t.Errorf("Balancer.New() err= %v", err)
		return nil
	}
	return b
}

func testCtrlSuit(t *testing.T, h http.Handler, req *http.Request, expectCode int, expectBody string) {
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	resp := rr.Result()
	if resp.StatusCode != expectCode {
		t.Errorf("Expect status code %d, but got %d", expectCode, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Read body err=%v", err)
		return
	}
	defer resp.Body.Close()

	if !bytes.Equal(body, []byte(expectBody)) {
		t.Errorf("Expect body '%s', but got '%s'", expectBody, string(body))
	}
}

func TestListAllVirtualServer(t *testing.T) {
	b := mockBalancer(t)
	h := ListAllVirtualServer(b)
	req := httptest.NewRequest("GET", "/vs", nil)
	expect := "Name:web, Address:127.0.0.1:8082, Status:stopped, Pool:\n127.0.0.1:10001, 127.0.0.1:10002\n\n"
	testCtrlSuit(t, h, req, 200, expect)
}

func TestListVirtualServer(t *testing.T) {
	b := mockBalancer(t)
	h := ListVirtualServer(b)
	req := httptest.NewRequest("GET", "/vs/web", nil)
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})

	expect := "127.0.0.1:10001, 127.0.0.1:10002"
	testCtrlSuit(t, h, req, 200, expect)
}

func TestStatsHandler(t *testing.T) {
	b := mockBalancer(t)
	h := &StatsHandler{b}
	req := httptest.NewRequest("GET", "/stats", nil)
	data := &stats.Data{
		StatusCode: "200",
		Method:     "POST",
		Path:       "test/",
		InBytes:    10,
		OutBytes:   20,
	}
	b.VServers[0].ServerStats["127.0.0.1:10001"] = stats.New()
	b.VServers[0].ServerStats["127.0.0.1:10002"] = stats.New()
	b.VServers[0].ServerStats["127.0.0.1:10001"].Inc(data)
	b.VServers[0].ServerStats["127.0.0.1:10001"].Inc(data)
	b.VServers[0].ServerStats["127.0.0.1:10002"].Inc(data)
	data.StatusCode = "500"
	b.VServers[0].ServerStats["127.0.0.1:10001"].Inc(data)
	expect := "Pool-web\n127.0.0.1:10001\nstatus_code: 200:2, 500:1\nmethod: POST:3\npath: test/:3\nrecv_bytes: 30\nsend_bytes: 60\n------\n127.0.0.1:10002\nstatus_code: 200:1\nmethod: POST:1\npath: test/:1\nrecv_bytes: 10\nsend_bytes: 20\n------"
	testCtrlSuit(t, h, req, 200, expect)
}

func TestModifyVirtualServerStatus(t *testing.T) {
	b := mockBalancer(t)
	h := ModifyVirtualServerStatus(b)

	// enable
	body, _ := json.Marshal(map[string]string{"action": "enable"})
	req := httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})
	expect := "success"
	t.Logf("Before enable: %s", b.VServers[0].Status())
	testCtrlSuit(t, h, req, 200, expect)
	time.Sleep(2 * time.Second)
	t.Logf("After enable: %s", b.VServers[0].Status())

	// repeat enable
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})
	testCtrlSuit(t, h, req, 200, "web is already enabled")

	// disalbe
	body, _ = json.Marshal(map[string]string{"action": "disable"})
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})
	testCtrlSuit(t, h, req, 200, expect)

	// unknown action
	body, _ = json.Marshal(map[string]string{})
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})
	testCtrlSuit(t, h, req, 400, "Unknown action")
}

func TestAddVirtualServer(t *testing.T) {
	b := mockBalancer(t)
	h := AddVirtualServer(b)
	body, _ := json.Marshal(map[string]string{"name": "redis", "address": "127.0.0.1:6379"})
	req := httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	expect := "Add success"

	testCtrlSuit(t, h, req, 200, expect)
	if len(b.VServers) != 2 {
		t.Errorf("Add virtual server fails")
		return
	}
	vs := b.VServers[1]
	if vs.Name != "redis" || vs.Address != "127.0.0.1:6379" {
		t.Errorf("Add virtual server fails")
	}
}

func TestAddPoolMember(t *testing.T) {
	b := mockBalancer(t)
	h := AddPoolMember(b)
	body, _ := json.Marshal(map[string]interface{}{"address": "127.0.0.1:10005", "weight": 1})
	req := httptest.NewRequest("POST", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})
	expect := "Add peer success"

	testCtrlSuit(t, h, req, 200, expect)

	if b.VServers[0].Pool.Size() != 3 {
		t.Errorf("Add peer fails")
	}

	// pool not exist
	req = httptest.NewRequest("POST", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "db",
	})
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNotFound.Error())
}

func TestDeletePoolMember(t *testing.T) {
	b := mockBalancer(t)
	h := DeletePoolMember(b)
	body, _ := json.Marshal(map[string]string{"address": "127.0.0.1:10001"})
	req := httptest.NewRequest("DELETE", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "web",
	})
	expect := "Remove peer success"

	testCtrlSuit(t, h, req, 200, expect)

	if b.VServers[0].Pool.Size() != 1 {
		t.Errorf("Remove peer fails")
	}

	req = httptest.NewRequest("DELETE", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{
		"name": "db",
	})
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNotFound.Error())
}
