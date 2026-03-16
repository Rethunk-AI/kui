package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						"request_id", requestIDFromContext(r),
						"error", recovered,
						"stack", string(debug.Stack()),
					)

					response := map[string]string{
						"error": "internal server error",
					}
					raw, _ := json.Marshal(response)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(raw)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

