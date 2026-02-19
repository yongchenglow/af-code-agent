package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// SecurityHeaders is a middleware that adds security headers to HTTP responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS filter in browsers
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy - restrict resource loading
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Cache control for sensitive data
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Remove server header
		w.Header().Del("Server")
		w.Header().Del("X-Powered-By")

		next.ServeHTTP(w, r)
	})
}

// HTTPSRedirect is a middleware that redirects HTTP requests to HTTPS
func HTTPSRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request is already HTTPS or behind a proxy with HTTPS
		if r.TLS != nil || isBehindTLSTerminationProxy(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Redirect to HTTPS
		host := r.Host
		if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}
		url := "https://" + host + r.URL.RequestURI()
		http.Redirect(w, r, url, http.StatusPermanentRedirect)
	})
}

// HSTS is a middleware that adds HTTP Strict Transport Security header
func HSTS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only add HSTS header over HTTPS
		if r.TLS != nil || isBehindTLSTerminationProxy(r) {
			// Max-age of 1 year (31536000 seconds)
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		next.ServeHTTP(w, r)
	})
}

// isBehindTLSTerminationProxy checks if the request is behind a TLS termination proxy
func isBehindTLSTerminationProxy(r *http.Request) bool {
	// Check for common headers set by load balancers and proxies
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	if r.Header.Get("X-Forwarded-Ssl") == "on" {
		return true
	}
	if r.Header.Get("X-Forwarded-Scheme") == "https" {
		return true
	}
	// Check for Front-End-Https header (used by some load balancers)
	if r.Header.Get("Front-End-Https") == "on" {
		return true
	}
	return false
}

// RequestID is a middleware that adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID already exists
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add to context for logging (would need context propagation)
		next.ServeHTTP(w, r)
	})
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	// Fallback to timestamp
	return time.Now().Format("20060102150405")
}

// Recovery is a middleware that recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				// Return 500 error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain applies multiple middleware in order
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// SecurityChain applies all security-related middleware
func SecurityChain(handler http.Handler) http.Handler {
	return Chain(
		handler,
		Recovery,
		RequestID,
		SecurityHeaders,
		HSTS,
	)
}

// ProductionChain applies all middleware for production environments
func ProductionChain(handler http.Handler) http.Handler {
	return Chain(
		handler,
		Recovery,
		RequestID,
		HTTPSRedirect,
		SecurityHeaders,
		HSTS,
	)
}
