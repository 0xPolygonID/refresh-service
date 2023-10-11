package server

import "net/http"

func internalError(w http.ResponseWriter, message error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(message.Error()))
}
