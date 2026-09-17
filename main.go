package main

import (
	"context"
	"errors"
	"fmt"
	"gateway/internal/config"
	"gateway/internal/jsonresponse"
	"gateway/internal/middleware"
	"gateway/internal/proxy"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	backendURL, err := cfg.App.Url()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	httpClient := &http.Client{
		Timeout: cfg.App.DialTimeout,
	}
	tokenCache := middleware.NewTokenCache()
	authService := middleware.NewAuth(
		tokenCache,
		backendURL,
		cfg.App.CheckRoute,
		httpClient,
	)

	proxyHandler := proxy.NewProxy(backendURL, cfg.App)
	publicHandler := middleware.LoggerMiddleware(proxyHandler)
	protectedHandler := middleware.LoggerMiddleware(authService.Middleware(proxyHandler))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		jsonresponse.WriteJSON(w, http.StatusBadGateway, jsonresponse.Body{
			Status:  "success",
			Message: "Alive",
		})
	})

	for _, v := range cfg.App.LoggedRoutes {
		mux.Handle(v, publicHandler)
	}

	for _, v := range cfg.App.ProtectedRoutes {
		mux.Handle(v, protectedHandler)
	}

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

	fmt.Println("Server is running on http://" + cfg.Server.Addr())

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
