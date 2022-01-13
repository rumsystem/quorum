package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

// NewJWTToken creates a new jwt token
func NewJWTToken(name string, jwtKey string, exp time.Time) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = name
	claims["exp"] = exp.Unix()

	return token.SignedString([]byte(jwtKey))
}

// IsJWTTokenExpires checks if the token is expired
func IsJWTTokenExpired(tokenStr string, jwtKey string) bool {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})

	if err != nil {
		e := err.(*jwt.ValidationError)
		if e.Errors == jwt.ValidationErrorExpired {
			return true
		}
	}

	return false
}

// IsJWTTokenValid checks if the token is valid, invalid include expired or invalid
func IsJWTTokenValid(tokenStr string, jwtKey string) (bool, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})

	if err != nil {
		e := err.(*jwt.ValidationError)
		if e.Errors == jwt.ValidationErrorExpired {
			return false, fmt.Errorf("token expires: %s", err)
		}

		return false, fmt.Errorf("token is invalid: %s", err)
	}

	return true, nil
}
