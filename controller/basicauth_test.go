package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func testSuit(t *testing.T, username, password string, expected_status int) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	authToken := &Authentication{Username: "admin", Password: "admin"}
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth(username, password)

	r := httptest.NewRecorder()
	h := BasicAuth(authToken)(handler)
	h.ServeHTTP(r, req)

	resp := r.Result()
	//t.Logf("%v", resp)
	if resp.StatusCode != expected_status {
		t.Errorf("expect %d, but got %d", expected_status, resp.StatusCode)
	}
}

func TestAuthPass(t *testing.T) {
	testSuit(t, "admin", "admin", 200)
}

func TestAuthFail(t *testing.T) {
	testSuit(t, "admin", "error", 401)
}
