package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

func HashedPassword(password string) (string, error) {

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
