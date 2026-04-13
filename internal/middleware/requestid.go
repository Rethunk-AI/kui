package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = newRequestID()
				r.Header.Set("X-Request-ID", requestID)
			}
			r = r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID))
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}

type requestIDContextKey struct{}

func newRequestID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}

func requestIDFromContext(r *http.Request) string {
	value := r.Context().Value(requestIDContextKey{})
	requestID, _ := value.(string)
	return requestID
}
