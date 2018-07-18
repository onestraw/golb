package retry

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
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
	if res.StatusCode != http.StatusOK {
		t.Errorf("Response Code should be %d, but got %d", http.StatusOK, res.StatusCode)
	}

	respBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if !bytes.Equal(respBody, RESPONSE) {
		t.Errorf("expected '%s', but got '%s'", RESPONSE, respBody)
	}
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
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Response Code should be %d, but got %d", http.StatusInternalServerError, res.StatusCode)
	}
}
