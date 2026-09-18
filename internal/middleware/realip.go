package middleware

import (
	"errors"
	"gateway/internal/jsonresponse"
	"log/slog"
	"net"
	"net/http"
)

type RealIP struct {
	behindProxy bool
}

func NewRealIP(behindProxy bool) *RealIP {
	return &RealIP{
		behindProxy: behindProxy,
	}
}

func (ipResolver *RealIP) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ipResolver.behindProxy {
			realIp := r.Header.Get("X-Real-IP")
			realPort := r.Header.Get("X-Real-Port")
			if realIp != "" && realPort != "" {
				r.RemoteAddr = net.JoinHostPort(realIp, realPort)
			} else {
				slog.Error(
					"proxy configuration error",
					"error", errors.New("BEHIND_PROXY set TRUE, but X-Real-IP/X-Real-Port is empty"),
					"path", r.URL.Path,
				)
				jsonresponse.WriteJSON(w, http.StatusInternalServerError, jsonresponse.Body{
					Status:  "error",
					Message: "Internal Server Error",
				})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
