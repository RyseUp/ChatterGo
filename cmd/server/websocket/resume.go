package websocket

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type ResumeStore struct {
	mu      sync.RWMutex
	tokens  map[string]*ResumeSession
	cleanup *time.Ticker
}

type ResumeSession struct {
	UserID        uint64
	LastMessageID uint64
	LastSeenAt    time.Time
	ConnectionID  string
}

func NewResumeStore() *ResumeStore {
	rs := &ResumeStore{
		tokens:  make(map[string]*ResumeSession),
		cleanup: time.NewTicker(5 * time.Minute),
	}

	// Cleanup expired tokens
	go func() {
		for range rs.cleanup.C {
			rs.cleanupExpired()
		}
	}()

	return rs
}

func (rs *ResumeStore) GenerateToken(userID uint64, connID string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.tokens[token] = &ResumeSession{
		UserID:       userID,
		LastSeenAt:   time.Now(),
		ConnectionID: connID,
	}

	return token
}

func (rs *ResumeStore) Get(token string) (*ResumeSession, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	session, ok := rs.tokens[token]
	return session, ok
}

func (rs *ResumeStore) UpdateLastMessage(token string, messageID uint64) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if session, ok := rs.tokens[token]; ok {
		session.LastMessageID = messageID
		session.LastSeenAt = time.Now()
	}
}

func (rs *ResumeStore) Remove(token string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.tokens, token)
}

func (rs *ResumeStore) cleanupExpired() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	expiry := time.Now().Add(-30 * time.Minute)
	for token, session := range rs.tokens {
		if session.LastSeenAt.Before(expiry) {
			delete(rs.tokens, token)
		}
	}
}

func (rs *ResumeStore) Stop() {
	rs.cleanup.Stop()
}
