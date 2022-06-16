package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// NewJWTToken creates a new jwt token
func NewJWTToken(name string, role string, allowGroups []string, jwtKey string, exp time.Time) (string, error) {
	if allowGroups == nil {
		allowGroups = []string{}
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = name
	claims["role"] = role
	claims["allowGroups"] = allowGroups
	claims["exp"] = exp.Unix()

	return token.SignedString([]byte(jwtKey))
}

// ParseJWTToken parse jwt token string to *jwt.Token
func ParseJWTToken(tokenStr string, jwtKey string) (*jwt.Token, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

// IsJWTTokenExpires checks if the token is expired
func IsJWTTokenExpired(tokenStr string, jwtKey string) bool {
	_, err := ParseJWTToken(tokenStr, jwtKey)
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
	_, err := ParseJWTToken(tokenStr, jwtKey)
	if err != nil {
		e := err.(*jwt.ValidationError)
		if e.Errors == jwt.ValidationErrorExpired {
			return false, fmt.Errorf("token expires: %s", err)
		}

		return false, fmt.Errorf("token is invalid: %s", err)
	}

	return true, nil
}
