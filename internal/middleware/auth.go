package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type authContextKey struct{}

type AuthenticatedUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type authClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	Role     string `json:"role"`
}

type AuthOptions struct {
	SkipExactPaths  []string
	SkipPrefixPaths []string
}

const SessionCookieName = "kui_session"

func UserFromContext(r *http.Request) (AuthenticatedUser, bool) {
	value := r.Context().Value(authContextKey{})
	user, ok := value.(AuthenticatedUser)
	return user, ok
}

func Auth(secret string, options AuthOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if shouldSkipAuth(path, options.SkipExactPaths, options.SkipPrefixPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if secret == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			rawToken, err := tokenFromRequest(r)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			claims := &authClaims{}
			token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
				method, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok || method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, fmt.Errorf("unexpected signing method: %T", token.Method)
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			user := AuthenticatedUser{
				ID:       claims.Subject,
				Username: claims.Username,
				Role:     claims.Role,
			}
			r = r.WithContext(context.WithValue(r.Context(), authContextKey{}, user))
			next.ServeHTTP(w, r)
		})
	}
}

func tokenFromRequest(r *http.Request) (string, error) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing token")
	}

	const bearer = "Bearer "
	if !strings.HasPrefix(authHeader, bearer) {
		return "", errors.New("malformed authorization header")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, bearer))
	if token == "" {
		return "", errors.New("missing token")
	}
	return token, nil
}

func shouldSkipAuth(path string, exacts, prefixes []string) bool {
	for _, exact := range exacts {
		if path == exact {
			return true
		}
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	payload := map[string]string{
		"error": message,
	}
	raw, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
