package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    Server
	Redis     Redis
	App       Backend
	RateLimit RateLimit
}

type HostWithPort struct {
	Host string
	Port int
}

type Server struct {
	HostWithPort
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type Redis struct {
	HostWithPort
	Password    string
	DialTimeout time.Duration
}

func (hp HostWithPort) Addr() string {
	return net.JoinHostPort(hp.Host, strconv.Itoa(hp.Port))
}

type Backend struct {
	Host            string
	Port            int
	DialTimeout     time.Duration
	ResponseTimeout time.Duration
	IdleTimeout     time.Duration
	MaxIdleConns    int
}

func (b Backend) Url() (*url.URL, error) {
	return url.Parse("http://" + net.JoinHostPort(b.Host, strconv.Itoa(b.Port)))
}

type RateLimit struct {
	RatePerWindow int
	WindowSecond  int
}

func Load() (Config, error) {
	var errs []error
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}

	server, err := loadServer()
	if err != nil {
		errs = append(errs, err)
	}

	redis, err := loadRedis()
	if err != nil {
		errs = append(errs, err)
	}

	app, err := loadApp()
	if err != nil {
		errs = append(errs, err)
	}

	rateLimit, err := loadRL()
	if err != nil {
		errs = append(errs, err)
	}

	config := Config{
		Server:    server,
		Redis:     redis,
		App:       app,
		RateLimit: rateLimit,
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	return config, nil
}

func getEnv(field, defaultValue string) string {
	value := os.Getenv(field)
	if value == "" {
		return defaultValue
	}
	return value
}

func mustGetEnv(field string) (string, error) {
	value, ok := os.LookupEnv(field)
	if !ok || value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return value, nil
}

func loadServer() (Server, error) {
	var errs []error

	host := getEnv("SERVER_HOST", "localhost")

	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid SERVER_PORT: %w", err))
	}

	readTimeout, err := time.ParseDuration(getEnv("READ_TIMEOUT", "5s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid READ_TIMEOUT: %w", err))
	}

	writeTimeout, err := time.ParseDuration(getEnv("WRITE_TIMEOUT", "10s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid WRITE_TIMEOUT: %w", err))
	}

	idleTimeout, err := time.ParseDuration(getEnv("IDLE_TIMEOUT", "120s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid IDLE_TIMEOUT: %w", err))
	}

	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "5s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err))
	}

	server := Server{
		Host: host, Port: port,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
	}

	if len(errs) > 0 {
		return Server{}, errors.Join(errs...)
	}
	return server, nil
}

func loadRedis() (Redis, error) {
	var errs []error

	host := getEnv("SERVER_HOST", "localhost")

	port, err := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid REDIS_PORT: %w", err))
	}

	password, err := mustGetEnv("REDIS_PWD")
	if err != nil {
		errs = append(errs, errors.New("REDIS_PWD must be init"))
	}

	timeout, err := time.ParseDuration(getEnv("REDIS_TIMEOUT", "30s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid REDIS_TIMEOUT: %w", err))
	}

	redis := Redis{
		Host: host, Port: port,
		Password:    password,
		DialTimeout: timeout,
	}

	if len(errs) > 0 {
		return Redis{}, errors.Join(errs...)
	}
	return redis, nil
}

func loadApp() (Backend, error) {
	var errs []error

	host, err := mustGetEnv("APP_HOST")
	if err != nil {
		errs = append(errs, errors.New("APP_HOST must be init"))
	}

	portStr, err := mustGetEnv("APP_PORT")
	if err != nil {
		errs = append(errs, errors.New("APP_PORT must be init"))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid APP_PORT: %w", err))
	}

	connectTimeout, err := time.ParseDuration(getEnv("APP_CONNECT_TIMEOUT", "10s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid APP_CONNECT_TIMEOUT: %w", err))
	}

	responseTimeout, err := time.ParseDuration(getEnv("APP_RESPONSE_TIMEOUT", "120s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid APP_RESPONSE_TIMEOUT: %w", err))
	}

	idleTimeout, err := time.ParseDuration(getEnv("APP_IDLE_TIMEOUT", "120s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid APP_IDLE_TIMEOUT: %w", err))
	}

	maxIdleConns, err := strconv.Atoi(getEnv("APP_MAX_IDLE_CONNS", "10"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid APP_MAX_IDLE_CONNS: %w", err))
	}

	app := Backend{
		Host:            host,
		Port:            port,
		DialTimeout:     connectTimeout,
		ResponseTimeout: responseTimeout,
		IdleTimeout:     idleTimeout,
		MaxIdleConns:    maxIdleConns,
	}

	if len(errs) > 0 {
		return Backend{}, errors.Join(errs...)
	}
	return app, nil
}

func loadRL() (RateLimit, error) {
	var errs []error

	rpw, err := strconv.Atoi(getEnv("RATE_LIMIT_RPW", "8080"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid RATE_LIMIT_RPW: %w", err))
	}

	windowSize, err := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SECONDS", "8080"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid RATE_LIMIT_WINDOW_SECONDS: %w", err))
	}

	rl := RateLimit{
		RatePerWindow: rpw,
		WindowSecond:  windowSize,
	}

	if len(errs) > 0 {
		return RateLimit{}, errors.Join(errs...)
	}
	return rl, nil
}
