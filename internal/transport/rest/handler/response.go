package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type errorResponse struct {
	Code    int
	Message string
}

// Respond is a function to send http responses.
func respond(w http.ResponseWriter, code int, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't marshal the given payload: %v", err), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't write response: %v", err), http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

// RespondErr is a function to make http error responses.
func respondErr(w http.ResponseWriter, code int, err error) {
	respErr := errorResponse{
		Code:    code,
		Message: err.Error(),
	}

	respond(w, code, respErr)
}
