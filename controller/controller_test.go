package controller

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onestraw/golb/balancer"
	"github.com/onestraw/golb/config"
	"github.com/onestraw/golb/stats"
)

func mockBalancer(t *testing.T) *balancer.Balancer {
	jsonBody := `{"virtual_server":[{"name":"web","address":"127.0.0.1:8082","server_name":"localhost","pool":[{"address":"127.0.0.1:10001","weight":1},{"address":"127.0.0.1:10002","weight":2}],"lb_method":"round-robin"}]}`
	c, err := config.LoadFromString(jsonBody)
	require.NoError(t, err)

	b, err := balancer.New(c.VServers)
	require.NoError(t, err)

	return b
}

func testCtrlSuit(t *testing.T, h http.Handler, req *http.Request, expectCode int, expectBody string) {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	resp := rr.Result()
	assert.Equal(t, expectCode, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, expectBody, string(body))
}

func TestController(t *testing.T) {
	c := New(&config.Controller{Address: "127.0.0.1:6587"})
	b := mockBalancer(t)
	c.Run(b)
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
	req = mux.SetURLVars(req, map[string]string{"name": "web"})

	expect := "127.0.0.1:10001, 127.0.0.1:10002"
	testCtrlSuit(t, h, req, 200, expect)

	req = mux.SetURLVars(req, map[string]string{"name": "not_exist"})
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNotFound.Error())
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
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	expect := "success"
	t.Logf("Before enable: %s", b.VServers[0].Status())
	testCtrlSuit(t, h, req, 200, expect)
	time.Sleep(time.Second)
	t.Logf("After enable: %s", b.VServers[0].Status())

	// repeat enable
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	testCtrlSuit(t, h, req, 200, "web is already enabled")

	// disalbe
	body, _ = json.Marshal(map[string]string{"action": "disable"})
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	testCtrlSuit(t, h, req, 200, expect)

	// virtual server not exist
	body, _ = json.Marshal(map[string]string{"action": "enable"})
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "not_exist"})
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNotFound.Error())

	// unknown action
	body, _ = json.Marshal(map[string]string{})
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	testCtrlSuit(t, h, req, 400, "Unknown action")

	// bad request
	req = httptest.NewRequest("POST", "/vs", strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	testCtrlSuit(t, h, req, 400, "EOF")
}

func TestAddVirtualServer(t *testing.T) {
	b := mockBalancer(t)
	h := AddVirtualServer(b)
	body, _ := json.Marshal(map[string]string{"name": "redis", "address": "127.0.0.1:6379"})
	req := httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	expect := "Add success"

	testCtrlSuit(t, h, req, 200, expect)

	assert.Equal(t, 2, len(b.VServers))
	vs := b.VServers[1]
	assert.Equal(t, "redis", vs.Name)
	assert.Equal(t, "127.0.0.1:6379", vs.Address)

	// test add fail
	body, _ = json.Marshal(map[string]string{"address": "127.0.0.1:6379"})
	req = httptest.NewRequest("POST", "/vs", bytes.NewReader(body))
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNameEmpty.Error())

	// test bad request
	req = httptest.NewRequest("POST", "/vs", strings.NewReader(""))
	testCtrlSuit(t, h, req, 400, "EOF")
}

func TestAddPoolMember(t *testing.T) {
	b := mockBalancer(t)
	h := AddPoolMember(b)
	body, _ := json.Marshal(map[string]interface{}{"address": "127.0.0.1:10005"})
	req := httptest.NewRequest("POST", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})

	testCtrlSuit(t, h, req, 200, "Add peer success")
	assert.Equal(t, 3, b.VServers[0].Pool.Size())

	// pool not exist
	req = httptest.NewRequest("POST", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "db"})
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNotFound.Error())

	// test bad request
	req = httptest.NewRequest("POST", "/vs/web/pool", strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	testCtrlSuit(t, h, req, 400, "EOF")
}

func TestDeletePoolMember(t *testing.T) {
	b := mockBalancer(t)
	h := DeletePoolMember(b)
	body, _ := json.Marshal(map[string]string{"address": "127.0.0.1:10001"})
	req := httptest.NewRequest("DELETE", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})

	testCtrlSuit(t, h, req, 200, "Remove peer success")
	assert.Equal(t, 1, b.VServers[0].Pool.Size())

	req = httptest.NewRequest("DELETE", "/vs/web/pool", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "db"})
	testCtrlSuit(t, h, req, 400, balancer.ErrVirtualServerNotFound.Error())

	// test bad request
	req = httptest.NewRequest("DELETE", "/vs/web/pool", strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"name": "web"})
	testCtrlSuit(t, h, req, 400, "EOF")
}
