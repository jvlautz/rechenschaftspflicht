package middlewares

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

func RequireBearerToken(bearerToken string) func(httprouter.Handle) httprouter.Handle {
	return func(h httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			prefixInvalid := !strings.EqualFold(parts[0], "Bearer")
			if prefixInvalid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tokenInvalid := parts[1] != bearerToken
			if tokenInvalid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			h(w, r, ps)
		}
	}
}
