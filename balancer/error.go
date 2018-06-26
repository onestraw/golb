package balancer

import (
	"errors"
	"net/http"
)

var (
	ErrNotSupportedMethod          = errors.New("Not supported LB method")
	ErrNotSupportedProto           = errors.New("Not supported Protocol")
	ErrVirtualServerNameEmpty      = errors.New("Vritual Server Name is not specified")
	ErrVirtualServerAddressEmpty   = errors.New("Vritual Server Address is not specified")
	ErrVirtualServerNameExisted    = errors.New("Vritual Server Name Existed")
	ErrVirtualServerAddressExisted = errors.New("Vritual Server Address Existed")
	ErrVirtualServerNotFound       = errors.New("Virtaul Server Not Found")
)

type BalancerError struct {
	StatusCode int
	ErrMsg     string
}

var (
	ErrBadRequest       = BalancerError{http.StatusBadRequest, "Reqeust Error"}
	ErrUnauthorized     = BalancerError{http.StatusUnauthorized, "Unauthorized"}
	ErrHostNotMatch     = BalancerError{http.StatusBadGateway, "Host Not Match"}
	ErrPeerNotFound     = BalancerError{http.StatusBadGateway, "Peer Not Found"}
	ErrInternalBalancer = BalancerError{http.StatusInternalServerError, "Balancer Internal Error"}
)

func WriteError(w http.ResponseWriter, err BalancerError) {
	w.WriteHeader(err.StatusCode)
	w.Write([]byte(err.ErrMsg))
}
