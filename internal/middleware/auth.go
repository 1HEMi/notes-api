package middleware

import (
	"context"
	"net/http"
	"notes/pkg/auth"
	"strings"
)

type key string

const userKey key = "user"

func JWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func GetUserID(ctx context.Context) int {
	if uid, ok := ctx.Value(userKey).(int); ok {
		return uid
	}
	return 0
}
