package controller

import "net/http"

type ControllerError struct {
	StatusCode int
	ErrMsg     string
}

func (ce *ControllerError) Error() string {
	return ce.ErrMsg
}

var (
	ErrUnauthorized  = &ControllerError{http.StatusUnauthorized, "Unauthorized"}
	ErrUnknownAction = &ControllerError{http.StatusBadRequest, "Unknown action"}
)

func WriteError(w http.ResponseWriter, err *ControllerError) {
	w.WriteHeader(err.StatusCode)
	w.Write([]byte(err.ErrMsg))
}

func WriteBadRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
}
