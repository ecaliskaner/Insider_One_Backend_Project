package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"golang.org/x/time/rate"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// LoggingMiddleware logs HTTP requests in structured JSON format
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a custom ResponseWriter to capture the status code
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start)
		logger.Info("HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.String("duration", duration.String()),
			slog.String("ip", getIP(r)),
		)
	})
}

// PanicRecoveryMiddleware recovers from panics and returns a 500 status
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic Recovered",
					slog.Any("error", err),
					slog.String("trace", string(debug.Stack())),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Internal Server Error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RateLimiterMiddleware prevents API abuse using a token bucket (100 req/sec)
func RateLimiterMiddleware(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(100), 200)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			logger.Warn("Rate Limit Exceeded", slog.String("ip", getIP(r)))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Too Many Requests"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getIP(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
