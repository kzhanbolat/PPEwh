package services

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type sessionRecord struct {
	AccountID string
	ExpiresAt time.Time
}

type SessionService struct {
	mu       sync.Mutex
	sessions map[string]sessionRecord
	ttl      time.Duration
}

func NewSessionService(ttl time.Duration) *SessionService {
	return &SessionService{
		sessions: make(map[string]sessionRecord),
		ttl:      ttl,
	}
}

func (s *SessionService) Create(accountID string) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(s.ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = sessionRecord{AccountID: accountID, ExpiresAt: expiresAt}
	return token, expiresAt, nil
}

func (s *SessionService) GetAccountID(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[token]
	if !ok {
		return "", false
	}
	if time.Now().After(rec.ExpiresAt) {
		delete(s.sessions, token)
		return "", false
	}
	return rec.AccountID, true
}

func (s *SessionService) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
