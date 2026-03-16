package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := requestIDFromContext(r)
			start := time.Now()

			writer := &statusCapturingResponseWriter{
				ResponseWriter: w,
			}

			next.ServeHTTP(writer, r)

			logger.Info(
				"http request",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", writer.status,
				"d", time.Since(start).String(),
			)
		})
	}
}

