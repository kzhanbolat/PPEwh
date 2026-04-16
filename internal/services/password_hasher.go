package services

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type PasswordHasher struct {
	pepper string
}

func NewPasswordHasher(keyPath string) (*PasswordHasher, error) {
	pepper, err := loadOrCreatePepper(keyPath)
	if err != nil {
		return nil, err
	}
	return &PasswordHasher{pepper: pepper}, nil
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	pw := strings.TrimSpace(password)
	if pw == "" {
		return "", errors.New("password is required")
	}
	sum, err := bcrypt.GenerateFromPassword([]byte(pw+h.pepper), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(sum), nil
}

func (h *PasswordHasher) Verify(hash, password string) bool {
	pw := strings.TrimSpace(password)
	if pw == "" || hash == "" {
		return false
	}
	// Preferred: peppered bcrypt.
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw+h.pepper)) == nil {
		return true
	}
	// Backward compatibility: old unpeppered bcrypt hashes.
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func loadOrCreatePepper(keyPath string) (string, error) {
	if b, err := os.ReadFile(keyPath); err == nil {
		pepper := strings.TrimSpace(string(b))
		if pepper != "" {
			return pepper, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		return "", err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	pepper := toHex(raw)
	if err := os.WriteFile(keyPath, []byte(pepper+"\n"), 0o600); err != nil {
		return "", err
	}
	return pepper, nil
}

func toHex(b []byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hex[v>>4]
		out[i*2+1] = hex[v&0x0f]
	}
	return string(out)
}
