package balancer

import (
	"errors"
	"net/http"
)

// Known errors.
var (
	ErrNotSupportedMethod          = errors.New("not supported LB method")
	ErrNotSupportedProto           = errors.New("not supported protocol")
	ErrVirtualServerNameEmpty      = errors.New("virtual server name is not specified")
	ErrVirtualServerAddressEmpty   = errors.New("virtual server address is not specified")
	ErrVirtualServerNameExisted    = errors.New("virtual server name existed")
	ErrVirtualServerAddressExisted = errors.New("virtual server address existed")
	ErrVirtualServerNotFound       = errors.New("virtual server not found")
)

type balancerError struct {
	StatusCode int
	ErrMsg     string
}

func (e *balancerError) Error() string {
	return e.ErrMsg
}

// Known balancerError.
var (
	ErrBadRequest       = &balancerError{http.StatusBadRequest, "Reqeust Error"}
	ErrHostNotMatch     = &balancerError{http.StatusBadRequest, "Host Not Match"}
	ErrPeerNotFound     = &balancerError{http.StatusBadGateway, "Peer Not Found"}
	ErrInternalBalancer = &balancerError{http.StatusInternalServerError, "Balancer Internal Error"}
)

// WriteError writes balancerError to http.ResponseWriter.
func WriteError(w http.ResponseWriter, err *balancerError) {
	w.WriteHeader(err.StatusCode)
	w.Write([]byte(err.ErrMsg))
}
