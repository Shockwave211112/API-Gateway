package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"gateway/internal/jsonresponse"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Auth struct {
	cache      *TokenCache
	backendURL *url.URL
	client     *http.Client
}

func NewAuth(cache *TokenCache, backendURL *url.URL, client *http.Client) *Auth {
	return &Auth{
		cache:      cache,
		backendURL: backendURL,
		client:     client,
	}
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unauthorized := func() {
			jsonresponse.WriteJSON(w, http.StatusUnauthorized, jsonresponse.Body{
				Status:  "error",
				Message: "Unauthorized",
			})
		}
		errorResponse := func() {
			jsonresponse.WriteJSON(w, http.StatusBadGateway, jsonresponse.Body{
				Status:  "error",
				Message: "Auth service unavailable",
			})
		}

		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			unauthorized()
			return
		}

		hashBytes := sha256.Sum256([]byte(token))
		hash := hex.EncodeToString(hashBytes[:])
		valid, found := a.cache.Get(hash)
		if !found {
			ok, err := a.verifyToken(r.Context(), token)
			if err != nil {
				slog.Error(
					"auth error (token verify)",
					"error", err.Error(),
					"path", r.URL.Path,
				)
				errorResponse()
				return
			}

			ttl := 10 * time.Minute
			if !ok {
				ttl = time.Hour
			}
			a.cache.Set(hash, ok, ttl)
			valid = ok
		}

		if !valid {
			unauthorized()
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Auth) verifyToken(ctx context.Context, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.backendURL.String()+"/user/info", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == http.StatusOK, nil
}
