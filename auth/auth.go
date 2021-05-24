package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/c-wiren/snackstoppen-backend/graph/model"
	"github.com/dgrijalva/jwt-go"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}
var Secret = "secret"

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
				return []byte(Secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "{\"errors\":[{\"message\": \"Invalid token\",\"extensions\": {\"code\": \"AUTHENTICATION_ERROR\"}}]}", http.StatusOK)
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				panic(fmt.Errorf("token claims error"))
			}
			rawID, _ := claims["id"].(float64)
			id := int(rawID)
			if err != nil {
				http.Error(w, "{\"errors\":[{\"message\": \"Invalid token\",\"extensions\": {\"code\": \"AUTHENTICATION_ERROR\"}}]}", http.StatusOK)
				return
			}
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

func CreateAccessToken(user *model.CompleteUser) *string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		//"username":  user.Username,
		//"firstname": user.Firstname,
		//"lastname":  user.Lastname,
		"id":   user.ID,
		"role": user.Role,
		//"email":     user.Email,
		//"image":     user.Image,
		//"created":   user.Created,
		//"logout":    user.Logout,
		"exp": time.Now().Add(time.Minute * 30).Unix(),
		"iat": time.Now().Unix(),
	})
	accessToken, _ := token.SignedString([]byte(Secret))
	return &accessToken
}

func CreateRefreshToken(user *model.CompleteUser) *string {
	token2 := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":     user.ID,
		"logout": user.Logout,
		"iat":    time.Now().Unix(),
	})
	refreshToken, _ := token2.SignedString([]byte(Secret))
	return &refreshToken
}

func CreateLoginResponse(user model.CompleteUser, includeRefreshToken bool) *model.LoginResponse {
	exp := time.Now().Add(time.Minute * 30)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		//"username":  user.Username,
		//"firstname": user.Firstname,
		//"lastname":  user.Lastname,
		"id":   user.ID,
		"role": user.Role,
		//"email":     user.Email,
		//"image":     user.Image,
		//"created":   user.Created,
		//"logout":    user.Logout,
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	})
	accessToken, _ := token.SignedString([]byte(Secret))

	var refreshToken string
	if includeRefreshToken {
		token2 := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"id":     user.ID,
			"logout": user.Logout,
			"iat":    time.Now().Unix(),
		})
		refreshToken, _ = token2.SignedString([]byte(Secret))
	}
	return &model.LoginResponse{User: &model.User{
		ID:        user.ID,
		Username:  user.Username,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Image:     user.Image,
	},
		Token:   accessToken,
		Refresh: &refreshToken,
		Expires: exp}
}
