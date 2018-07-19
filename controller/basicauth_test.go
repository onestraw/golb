package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, expected_status, resp.StatusCode)
}

func TestAuthPass(t *testing.T) {
	testSuit(t, "admin", "admin", 200)
}

func TestAuthFail(t *testing.T) {
	testSuit(t, "admin", "error", 401)
}
