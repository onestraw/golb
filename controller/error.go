package controller

import "net/http"

type ControllerError struct {
	StatusCode int
	ErrMsg     string
}

var (
	ErrUnauthorized = ControllerError{http.StatusUnauthorized, "Unauthorized"}
)

func WriteError(w http.ResponseWriter, err ControllerError) {
	w.WriteHeader(err.StatusCode)
	w.Write([]byte(err.ErrMsg))
}

func WriteBadRequest(w http.ResponseWriter, err error) {
	WriteError(w, ControllerError{http.StatusBadRequest, err.Error()})
}
