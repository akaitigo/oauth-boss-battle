package middleware

import (
	"net/http"
	"os"
	"strings"
)

// allowedOrigins returns the set of permitted CORS origins.
func allowedOrigins() map[string]struct{} {
	raw := os.Getenv("CORS_ORIGINS")
	if raw == "" {
		raw = "http://localhost:3000,http://localhost:5173"
	}
	origins := make(map[string]struct{})
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = struct{}{}
		}
	}
	return origins
}

// CORS wraps a handler with CORS headers using an origin whitelist.
func CORS(next http.Handler) http.Handler {
	origins := allowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := origins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
