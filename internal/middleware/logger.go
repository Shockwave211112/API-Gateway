package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
	"uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(status int) {
	sr.status = status
	sr.ResponseWriter.WriteHeader(status)
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		ctx := WithRequestID(r.Context(), reqID)
		r = r.WithContext(ctx)
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		start := time.Now()

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", duration.Milliseconds(),
			"request_id", reqID,
			"remote_addr", r.RemoteAddr,
		)
	})
}
