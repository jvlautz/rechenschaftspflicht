package middlewares

import (
	"fmt"
	"net/http"
)

// SecurityHeaders adds security headers to all HTTP responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cspNonce := GetNonceFromContext(r.Context())
		nonceDirective := ""
		if cspNonce != "" {
			nonceDirective = fmt.Sprintf(" 'nonce-%s'", cspNonce)
		}

		// Content Security Policy
		// Allow unpkg.com for CSS, self for scripts, and restrict everything else
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"style-src 'self' https://unpkg.com"+nonceDirective+"; "+
				"script-src 'self' https://cdn.jsdelivr.net; "+
				"img-src 'self'; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS Protection (legacy but still useful for older browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
