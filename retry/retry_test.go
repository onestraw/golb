package retry

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProxyRetry(t *testing.T, status_code int) {
	var RESPONSE = []byte("message from server")
	var count_fail = 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count_fail += 1
		if count_fail < TRY {
			t.Logf("%dth, simulate server code %d", count_fail, status_code)
			w.WriteHeader(status_code)
		}
		w.Header().Add("Content-Length", "20")
		w.Write(RESPONSE)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	H := Retry(handler)
	H.ServeHTTP(rr, req)

	res := rr.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	respBody, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, respBody, RESPONSE)
}

func TestProxyRetry500(t *testing.T) {
	testProxyRetry(t, http.StatusInternalServerError)
}

func TestProxyRetry502(t *testing.T) {
	testProxyRetry(t, http.StatusBadGateway)
}

func TestProxyRetry503(t *testing.T) {
	testProxyRetry(t, http.StatusServiceUnavailable)
}

func TestProxyRetry504(t *testing.T) {
	testProxyRetry(t, http.StatusGatewayTimeout)
}

func TestProxyRetryFail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	H := Retry(handler)
	H.ServeHTTP(rr, req)

	res := rr.Result()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}
