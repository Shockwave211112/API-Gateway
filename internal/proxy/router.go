package proxy

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/jsonresponse"
	"gateway/internal/middleware"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type contextKey string

const proxyErrorKey contextKey = "proxy_error"

func NewProxy(backendUrl *url.URL, app config.Backend, behindProxy bool) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			if behindProxy {
				realIp := pr.In.Header.Get("X-Real-IP")
				if realIp != "" {
					pr.In.RemoteAddr = realIp + ":0"
				} else {
					err := errors.New("BEHIND_PROXY set TRUE, but X-Real-IP is empty")
					ctx := context.WithValue(pr.In.Context(), proxyErrorKey, err)
					cancelCtx, cancel := context.WithCancel(ctx)
					cancel()

					pr.Out = pr.Out.WithContext(cancelCtx)
					return
				}
			}

			pr.Out.URL.Scheme = backendUrl.Scheme
			pr.Out.URL.Host = backendUrl.Host

			pr.Out.Header.Del("X-Forwarded-For")
			pr.SetXForwarded()

			if reqId, ok := middleware.RequestIDFromContext(pr.In.Context()); ok {
				pr.Out.Header.Set("X-Request-ID", reqId)
			}
		},

		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: app.DialTimeout,
			}).DialContext,
			ResponseHeaderTimeout: app.ResponseTimeout,
			IdleConnTimeout:       app.IdleTimeout,
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, proxyErr error) {
			if ctxErr, ok := r.Context().Value(proxyErrorKey).(error); ok {
				slog.Error(
					"request blocked by gateway",
					"reason", ctxErr,
				)
			} else {
				slog.Error(
					"proxy error",
					"error", proxyErr.Error(),
					"path", r.URL.Path,
				)
			}

			jsonresponse.WriteJSON(w, http.StatusBadGateway, jsonresponse.Body{
				Status:  "error",
				Message: "Internal Server Error",
			})
		},
	}

	return proxy
}
