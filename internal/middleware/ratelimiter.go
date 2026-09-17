package middleware

import (
	"fmt"
	"gateway/internal/jsonresponse"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redis  *redis.Client
	limit  int
	window time.Duration
	script *redis.Script
}

func NewRateLimiter(redisClient *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		limit:  limit,
		window: window,
		script: redis.NewScript(atomicLua),
	}
}

const atomicLua = `
	local current = redis.call("INCR", KEYS[1])
	if current == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	return current
	`

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		windowNumber := int(time.Now().Unix() / int64(rl.window.Seconds()))
		ip, err := netip.ParseAddrPort(r.RemoteAddr)
		if err != nil {
			slog.Error(
				"rate limiter error",
				"error", err.Error(),
				"path", r.URL.Path,
			)
			jsonresponse.WriteJSON(w, http.StatusBadGateway, jsonresponse.Body{
				Status:  "error",
				Message: "Internal Server Error",
			})
			return
		}
		key := fmt.Sprintf("ratelimit:%s:%d", ip.Addr(), windowNumber)

		result, err := rl.script.Run(r.Context(), rl.redis, []string{key}, rl.window.Seconds()).Int()
		if err != nil {
			slog.Error(
				"rate limiter error",
				"error", err.Error(),
				"path", r.URL.Path,
			)
			next.ServeHTTP(w, r) // fail open
			return
		}

		if result > rl.limit {
			reqId, ok := RequestIDFromContext(r.Context())
			if !ok {
				reqId = "unknown"
			}

			slog.Warn(
				"rate limit exceeded",
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", reqId,
				"remote_addr", r.RemoteAddr,
			)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			jsonresponse.WriteJSON(w, http.StatusTooManyRequests, jsonresponse.Body{
				Status:  "error",
				Message: "Please slow down",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
