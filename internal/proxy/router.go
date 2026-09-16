package proxy

import (
	"fmt"
	"gateway/internal/config"
	"gateway/internal/jsonresponse"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"uuid"
)

func NewProxy(backendUrl *url.URL, app config.Backend) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = backendUrl.Scheme
			pr.Out.URL.Host = backendUrl.Host

			pr.Out.Header.Del("X-Forwarded-For")
			pr.SetXForwarded()

			pr.Out.Header.Set("X-Request-ID", uuid.New().String())
		},

		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: app.DialTimeout,
			}).DialContext,
			ResponseHeaderTimeout: app.ResponseTimeout,
			IdleConnTimeout:       app.IdleTimeout,
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, proxyErr error) {
			fmt.Println(proxyErr.Error())
			jsonresponse.WriteJSON(w, http.StatusBadGateway, jsonresponse.Body{
				Status:  "error",
				Message: "Internal Server Error",
			})
		},
	}

	return proxy
}
