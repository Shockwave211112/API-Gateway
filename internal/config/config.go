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
	Host        string
	Port        int
	DialTimeout time.Duration
}

func (b Backend) Url() (*url.URL, error) {
	return url.Parse("http//" + net.JoinHostPort(b.Host, strconv.Itoa(b.Port)))
}

type RateLimit struct {
	RatePerWindow int
	WindowSecond  int
}

//func Load() (*Config, error) {
//
//}
