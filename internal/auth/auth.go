package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	// TokenTypeAccess -
	TokenTypeAccess TokenType = "chirpy-access"
)

func HashPassword(password string) (string, error) {

	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("error: %w", err)
	}

	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	is_pass, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("error: %w", err)
	}
	return is_pass, nil
}

// MakeJWT -
func MakeJWT(
	userID uuid.UUID,
	tokenSecret string,
	expiresIn time.Duration,
) (string, error) {
	signingKey := []byte(tokenSecret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    string(TokenTypeAccess),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	})
	return token.SignedString(signingKey)
}

// ValidateJWT -
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claimsStruct := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil },
	)
	if err != nil {
		return uuid.Nil, err
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("invalid issuer")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	prefix := "Bearer "
	token, ok := headers["Authorization"]
	if !ok {
		return "", errors.New("no token found")
	}
	for _, value := range token {
		if strings.Contains(value, prefix) {
			return strings.TrimPrefix(value, prefix), nil
		}
	}
	return "", errors.New("something went wrong")
}
