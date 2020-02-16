package main

import (
	"encoding/json"
	"net/http"
)

type responseErrorStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func responseError(w http.ResponseWriter, err error) {
	resp := &responseErrorStruct{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}

	str, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, string(str), http.StatusInternalServerError)
	}
}

func responseBadRequest(w http.ResponseWriter, err error) {
	resp := &responseErrorStruct{
		Code:    http.StatusBadRequest,
		Message: err.Error(),
	}

	str, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		http.Error(w, string(str), http.StatusBadRequest)
	}
}
