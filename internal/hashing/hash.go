package hashing

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gelozr/go-dash/internal/config"
)

type Driver string

const (
	Bcrypt   = Driver("bcrypt")
	Argon2ID = Driver("argon2id")
)

type Hasher interface {
	Hash(password string) (string, error)
	Check(password, hash string) (bool, error)
}

type Manager interface {
	Hasher
	RegisterHasher(d Driver, hasher Hasher) error
}

type manager struct {
	mu            sync.RWMutex
	hashers       map[Driver]Hasher
	defaultDriver Driver
}

func NewManager(cfg *config.Config) Manager {
	hashers := make(map[Driver]Hasher)
	hashers[Bcrypt] = NewBcryptHasher()
	hashers[Argon2ID] = NewArgon2IDHasher()

	return &manager{
		hashers:       hashers,
		defaultDriver: getDefaultDriver(cfg),
	}
}

func (m *manager) getDefaultHasher() (Hasher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if hasher, ok := m.hashers[m.defaultDriver]; ok {
		return hasher, nil
	}
	err := fmt.Errorf("hasher not found for driver %s", m.defaultDriver)
	return nil, fmt.Errorf("get hasher: %w", err)
}

func (m *manager) RegisterHasher(d Driver, hasher Hasher) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.hashers[d]
	if ok {
		return errors.New("hash driver already registered")
	}

	m.hashers[d] = hasher
	return nil
}

// Hash hashes a plaintext
func (m *manager) Hash(password string) (string, error) {
	hasher, err := m.getDefaultHasher()
	if err != nil {
		return "", err
	}

	return hasher.Hash(password)
}

// Check checks if the given password matches the hashed password
func (m *manager) Check(password, hash string) (bool, error) {
	hasher, err := m.getDefaultHasher()
	if err != nil {
		return false, err
	}

	return hasher.Check(password, hash)
}

func getDefaultDriver(cfg *config.Config) Driver {
	defaultHasher := Bcrypt
	if cfg.HashingDriver != "" {
		defaultHasher = Driver(cfg.HashingDriver)
	}
	return defaultHasher
}
