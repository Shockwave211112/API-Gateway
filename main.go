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
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
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

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	httpClient := newHTTPClient(cfg.App)
	redisClient := newRedisClient(cfg.Redis)
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("failed to close redis client",
				"error", err,
			)
		}
	}()

	public, protected := buildHandlers(cfg, backendURL, httpClient, redisClient)
	mux := buildMux(cfg.App, public, protected)

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	if err := runServer(server, cfg.Server.ShutdownTimeout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newHTTPClient(cfg config.Backend) *http.Client {
	return &http.Client{
		Timeout: cfg.DialTimeout,
	}
}

func newRedisClient(cfg config.Redis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        cfg.Addr(),
		Password:    cfg.Password,
		DialTimeout: cfg.DialTimeout,
	})
}

func buildHandlers(cfg config.Config, backendURL *url.URL, httpClient *http.Client, redisClient *redis.Client) (public, protected http.Handler) {
	tokenCache := middleware.NewTokenCache()
	authService := middleware.NewAuth(
		tokenCache,
		backendURL,
		cfg.App.CheckRoute,
		httpClient,
	)
	rateLimitService := middleware.NewRateLimiter(
		redisClient,
		cfg.RateLimit.RatePerWindow,
		cfg.RateLimit.WindowSize,
	)
	realIpService := middleware.NewRealIP(cfg.Server.BehindProxy)

	proxyHandler := proxy.NewProxy(backendURL, cfg.App)
	public = middleware.LoggerMiddleware(
		realIpService.Middleware(
			rateLimitService.Middleware(proxyHandler)))
	protected = middleware.LoggerMiddleware(
		realIpService.Middleware(
			rateLimitService.Middleware(
				authService.Middleware(proxyHandler))))

	return public, protected
}

func buildMux(cfg config.Backend, public, protected http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		jsonresponse.WriteJSON(w, http.StatusOK, jsonresponse.Body{
			Status:  "success",
			Message: "Alive",
		})
	})

	for _, v := range cfg.LoggedRoutes {
		mux.Handle(v, public)
	}

	for _, v := range cfg.ProtectedRoutes {
		mux.Handle(v, protected)
	}

	return mux
}

func runServer(server *http.Server, shutdownTimeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	fmt.Println("Server is running on http://" + server.Addr)

	select {
	case <-ctx.Done():
		fmt.Println("Shutting down server gracefully...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("forced server shutdown: %w", err)
		}

		fmt.Println("Server stopped successfully")
		return nil

	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}
