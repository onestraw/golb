package balancer

import (
	"errors"
	"net/http"
)

var (
	ErrNotSupportedMethod = errors.New("Not supported LB method")
	ErrNotSupportedProto  = errors.New("Not supported Protocol")
)

type BalancerError struct {
	StatusCode int
	ErrMsg     string
}

var (
	ErrUnauthorized     = BalancerError{http.StatusUnauthorized, "Unauthorized"}
	ErrHostNotMatch     = BalancerError{http.StatusBadGateway, "Host Not Match"}
	ErrPeerNotFound     = BalancerError{http.StatusBadGateway, "Peer Not Found"}
	ErrInternalBalancer = BalancerError{http.StatusInternalServerError, "Balaner Internal Error"}
)

func WriteError(w http.ResponseWriter, err BalancerError) {
	w.WriteHeader(err.StatusCode)
	w.Write([]byte(err.ErrMsg))
}
