package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

// AuthMiddleware creates a middleware for Firebase authentication
func AuthMiddleware(authClient *auth.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorizationHeader := r.Header.Get("Authorization")
			if authorizationHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authorizationHeader, "Bearer ")
			if token == authorizationHeader {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenVerified, err := authClient.VerifyIDToken(r.Context(), token)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add the verified token to the request context
			ctx := context.WithValue(r.Context(), "user", tokenVerified)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the context
func GetUserID(ctx context.Context) (string, bool) {
	token, ok := ctx.Value("user").(*auth.Token)
	if !ok {
		return "", false
	}
	return token.UID, true
}
