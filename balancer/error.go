package balancer

import (
	"errors"
	"net/http"
)

// Known errors.
var (
	ErrNotSupportedMethod          = errors.New("Not supported LB method")
	ErrNotSupportedProto           = errors.New("Not supported Protocol")
	ErrVirtualServerNameEmpty      = errors.New("Vritual Server Name is not specified")
	ErrVirtualServerAddressEmpty   = errors.New("Vritual Server Address is not specified")
	ErrVirtualServerNameExisted    = errors.New("Vritual Server Name Existed")
	ErrVirtualServerAddressExisted = errors.New("Vritual Server Address Existed")
	ErrVirtualServerNotFound       = errors.New("Virtaul Server Not Found")
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
