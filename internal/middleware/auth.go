package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"gateway/internal/jsonresponse"
	"net/http"
	"net/url"
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
		token := r.Header.Get("Authorization")
		if token == "" || token == "Bearer " {
			jsonresponse.WriteJSON(w, http.StatusUnauthorized, jsonresponse.Body{
				Status:  "error",
				Message: "Unauthorized",
			})
			return
		}

		hashBytes := sha256.Sum256([]byte(token))
		hash := hex.EncodeToString(hashBytes[:])
		valid, found := a.cache.Get(hash)
		if !found {
			req, _ := http.NewRequest("GET", a.backendURL.String()+"/user/info", nil)
			req.Header.Set("Authorization", token)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			res, _ := a.client.Do(req)
			if res.StatusCode != http.StatusOK {
				a.cache.Set(hash, false, time.Hour)
				valid = false
			} else {
				a.cache.Set(hash, true, 10*time.Minute)
				valid = true
			}
		}
		if !valid {
			jsonresponse.WriteJSON(w, http.StatusUnauthorized, jsonresponse.Body{
				Status:  "error",
				Message: "Unauthorized",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
