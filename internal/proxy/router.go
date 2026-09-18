package proxy

import (
	"gateway/internal/config"
	"gateway/internal/jsonresponse"
	"gateway/internal/middleware"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewProxy(backendUrl *url.URL, app config.Backend) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
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
			MaxIdleConns:          app.MaxIdleConns,
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, proxyErr error) {
			slog.Error(
				"proxy error",
				"error", proxyErr.Error(),
				"path", r.URL.Path,
			)

			jsonresponse.WriteJSON(w, http.StatusBadGateway, jsonresponse.Body{
				Status:  "error",
				Message: "Internal Server Error",
			})
		},
	}

	return proxy
}
