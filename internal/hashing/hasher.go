package hashing

import (
	"errors"
	"fmt"

	"github.com/alexedwards/argon2id"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func NewBcryptHasher() BcryptHasher {
	return BcryptHasher{}
}

var _ Hasher = (*BcryptHasher)(nil)

func (BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %v", err)
	}
	return string(bytes), nil
}

func (BcryptHasher) Check(password string, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, fmt.Errorf("bcrypt compare password hash: %w", err)
		}
	}

	return true, nil
}

type Argon2IDHasher struct{}

func NewArgon2IDHasher() Argon2IDHasher {
	return Argon2IDHasher{}
}

var _ Hasher = (*Argon2IDHasher)(nil)

func (Argon2IDHasher) Hash(password string) (string, error) {
	s, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("argon hash password: %w", err)
	}
	return s, nil
}

func (Argon2IDHasher) Check(password, hash string) (bool, error) {
	ok, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("argon compare password hash: %w", err)
	}
	return ok, nil
}
