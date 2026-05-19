package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type requestIDContextKey struct{}

const RequestIDHeader = "X-Request-ID"

// RequestIDFromContext returns the request ID stored by RequestIDMiddleware.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// RequestIDMiddleware preserves an inbound request ID or creates one for tracing.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(RequestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
			slog.Int64("duration_ms", duration.Milliseconds()),
			slog.String("ip", getIP(r)),
			slog.String("request_id", RequestIDFromContext(r.Context())),
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
				if _, writeErr := w.Write([]byte(`{"error": "Internal Server Error"}`)); writeErr != nil {
					logger.Error("Failed to write panic response", slog.Any("error", writeErr))
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterMiddleware prevents API abuse using a per-client token bucket.
func RateLimiterMiddleware(next http.Handler) http.Handler {
	var mu sync.Mutex
	clients := make(map[string]*clientLimiter)
	lastCleanup := time.Now()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if now.Sub(lastCleanup) > time.Minute {
			for clientIP, client := range clients {
				if now.Sub(client.lastSeen) > 3*time.Minute {
					delete(clients, clientIP)
				}
			}
			lastCleanup = now
		}

		client, ok := clients[ip]
		if !ok {
			client = &clientLimiter{
				limiter: rate.NewLimiter(rate.Limit(100), 200),
			}
			clients[ip] = client
		}
		client.lastSeen = now
		return client.limiter
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		limiter := getLimiter(ip)
		if !limiter.Allow() {
			logger.Warn("Rate Limit Exceeded", slog.String("ip", ip))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			if _, writeErr := w.Write([]byte(`{"error": "Too Many Requests"}`)); writeErr != nil {
				logger.Error("Failed to write rate-limit response", slog.Any("error", writeErr))
			}
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
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		if ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(bytes[:])
}
