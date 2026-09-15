package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type infoHandler struct {
	response string
}

func (ih *infoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request Accepted")

	data := Response{
		Status:  "success",
		Message: ih.response,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonBytes)
	if err != nil {
		return
	}
}

func main() {
	mux := http.NewServeMux()

	mainHandler := &infoHandler{response: "OK"}

	mux.Handle("GET /health", mainHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("Server is running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
