package config

import (
	"net"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Server    Server
	Redis     Redis
	App       Backend
	RateLimit RateLimit
}

type Server struct {
	HostWithPort
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	BehindProxy     bool
}

type RateLimit struct {
	RatePerWindow int
	WindowSize    time.Duration
}

type Redis struct {
	HostWithPort
	Password    string
	DialTimeout time.Duration
}

type Backend struct {
	Host            string
	Port            int
	DialTimeout     time.Duration
	ResponseTimeout time.Duration
	IdleTimeout     time.Duration
	MaxIdleConns    int
	CheckRoute      string
	LoggedRoutes    []string
	ProtectedRoutes []string
}

func (b Backend) Url() (*url.URL, error) {
	return url.Parse("http://" + net.JoinHostPort(b.Host, strconv.Itoa(b.Port)))
}

type HostWithPort struct {
	Host string
	Port int
}

func (hp HostWithPort) Addr() string {
	return net.JoinHostPort(hp.Host, strconv.Itoa(hp.Port))
}
