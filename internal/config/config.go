package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

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

func loadServer() (Server, error) {
	var c errCollector

	host := getEnv("SERVER_HOST", "localhost")
	port := c.intField("SERVER_PORT", getEnv("SERVER_PORT", "8080"))
	readTimeout := c.duration("READ_TIMEOUT", getEnv("READ_TIMEOUT", "5s"), time.Second)
	writeTimeout := c.duration("WRITE_TIMEOUT", getEnv("WRITE_TIMEOUT", "10s"), time.Second)
	idleTimeout := c.duration("IDLE_TIMEOUT", getEnv("IDLE_TIMEOUT", "120s"), time.Second)
	shutdownTimeout := c.duration("SHUTDOWN_TIMEOUT", getEnv("SHUTDOWN_TIMEOUT", "5s"), time.Second)
	behindProxy := getEnv("BEHIND_REVERSE_PROXY", "false") == "true"

	if err := c.err(); err != nil {
		return Server{}, err
	}

	return Server{
		Host: host, Port: port,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
		BehindProxy:     behindProxy,
	}, nil
}

func loadRedis() (Redis, error) {
	var c errCollector

	host := getEnv("REDIS_HOST", "localhost")
	port := c.intField("REDIS_PORT", getEnv("REDIS_PORT", "6379"))
	password := getEnv("REDIS_PWD", "")
	timeout := c.duration("REDIS_TIMEOUT", getEnv("REDIS_TIMEOUT", "30s"), time.Second)

	if err := c.err(); err != nil {
		return Redis{}, err
	}

	return Redis{
		Host: host, Port: port,
		Password:    password,
		DialTimeout: timeout,
	}, nil
}

func loadApp() (Backend, error) {
	var c errCollector

	host := c.required("APP_HOST")
	port := c.intField("APP_PORT", c.required("APP_PORT"))
	connectTimeout := c.duration("APP_CONNECT_TIMEOUT", getEnv("APP_CONNECT_TIMEOUT", "10s"), time.Second)
	responseTimeout := c.duration("APP_RESPONSE_TIMEOUT", getEnv("APP_RESPONSE_TIMEOUT", "20s"), time.Second)
	idleTimeout := c.duration("APP_IDLE_TIMEOUT", getEnv("APP_IDLE_TIMEOUT", "120s"), time.Second)
	maxIdleConns := c.intField("APP_MAX_IDLE_CONNS", getEnv("APP_MAX_IDLE_CONNS", "10"))
	loggedRoutes := strings.Split(getEnv("APP_LOGGED_ROUTES", ""), ",")
	for i := range loggedRoutes {
		loggedRoutes[i] = strings.TrimSpace(loggedRoutes[i])
	}
	protectedRoutes := strings.Split(getEnv("APP_PROTECTED_ROUTES", ""), ",")
	for i := range protectedRoutes {
		protectedRoutes[i] = strings.TrimSpace(protectedRoutes[i])
	}
	checkRoute := c.required("APP_CHECK_ROUTE")

	if err := c.err(); err != nil {
		return Backend{}, err
	}

	return Backend{
		Host:            host,
		Port:            port,
		DialTimeout:     connectTimeout,
		ResponseTimeout: responseTimeout,
		IdleTimeout:     idleTimeout,
		MaxIdleConns:    maxIdleConns,
		LoggedRoutes:    loggedRoutes,
		ProtectedRoutes: protectedRoutes,
		CheckRoute:      checkRoute,
	}, nil
}

func loadRL() (RateLimit, error) {
	var c errCollector

	rpw := c.intField("RATE_LIMIT_RPW", getEnv("RATE_LIMIT_RPW", "10"))
	windowSize := c.duration("RATE_LIMIT_WINDOW", getEnv("RATE_LIMIT_WINDOW", "1s"), time.Second)

	if err := c.err(); err != nil {
		return RateLimit{}, err
	}

	return RateLimit{
		RatePerWindow: rpw,
		WindowSize:    windowSize,
	}, nil
}
