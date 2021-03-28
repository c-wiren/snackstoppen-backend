package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// A stand-in for our database backed user object
type User struct {
	ID   int
	Role string
}

// Middleware decodes the share session cookie and packs the session into context
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := r.Header.Get("authorization")
			if rawToken == "" {
				next.ServeHTTP(w, r)
				return
			}
			splitToken := strings.Split(rawToken, "Bearer ")
			if len(splitToken) != 2 {
				return
			}

			// Parse JWT
			token, err := jwt.Parse(splitToken[1], func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte("secret"), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "{\"errors\":[{\"message\": \"Invalid token\"}]}", http.StatusForbidden)
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				panic(fmt.Errorf("token claims error"))
			}
			id, _ := claims["id"].(int)
			role, _ := claims["role"].(string)

			// put it in context
			user := User{ID: id, Role: role}
			ctx := context.WithValue(r.Context(), userCtxKey, &user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *User {
	raw, _ := ctx.Value(userCtxKey).(*User)
	return raw
}
