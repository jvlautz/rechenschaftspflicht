package middlewares

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/a-h/templ"
)

type nonceContextKey struct{}

// TemplCSSWithNonce generates a CSP nonce per request and stores it in the
// request context. templ.WithNonce ensures all inline <style> tags rendered
// by templ include the nonce attribute. SecurityHeaders must run after this
// middleware to pick up the nonce for the CSP header.
func TemplCSSWithNonce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonceBytes := make([]byte, 16)
		if _, err := rand.Read(nonceBytes); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		nonce := base64.StdEncoding.EncodeToString(nonceBytes)

		ctx := context.WithValue(r.Context(), nonceContextKey{}, nonce)
		ctx = templ.WithNonce(ctx, nonce)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetNonceFromContext returns the CSP nonce stored in the request context,
// or empty string if not present.
func GetNonceFromContext(ctx context.Context) string {
	nonce, _ := ctx.Value(nonceContextKey{}).(string)
	return nonce
}
