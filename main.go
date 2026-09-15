package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/internal/config"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request Accepted")

		data := Response{
			Status:  "success",
			Message: "OK",
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
	})

	errChan := make(chan error, 1)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	fmt.Println("Server is running on http://localhost:8080")

	select {
	case <-ctx.Done():
		fmt.Println("Stop via commandline")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Error on server shutdown: %v\n", err)
			stop()
			os.Exit(1)
		}

		fmt.Println("Server successfully stopped")
	case err := <-errChan:
		fmt.Printf("Server error: %v\n", err)
		stop()
		os.Exit(1)
	}
}
