package controller

import "net/http"

type controllerError struct {
	StatusCode int
	ErrMsg     string
}

func (e *controllerError) Error() string {
	return e.ErrMsg
}

// Known controllerError.
var (
	ErrUnauthorized  = &controllerError{http.StatusUnauthorized, "Unauthorized"}
	ErrUnknownAction = &controllerError{http.StatusBadRequest, "Unknown action"}
)

// WriteError writes the controllerError to http.ResponseWriter.
func WriteError(w http.ResponseWriter, err *controllerError) {
	w.WriteHeader(err.StatusCode)
	w.Write([]byte(err.ErrMsg))
}

// WriteBadRequest writes the error with 400 to http.ResponseWriter.
func WriteBadRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
}
