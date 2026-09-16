package jsonresponse

import (
	"encoding/json"
	"net/http"
)

type Body struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, body Body) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonBytes)
}
