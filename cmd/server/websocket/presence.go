package websocket

import "sync"

type PresenceRegistry struct {
	mu     sync.RWMutex
	online map[uint64]int
}

func NewPresenceRegistry() *PresenceRegistry {
	return &PresenceRegistry{
		online: make(map[uint64]int),
	}
}

func (p *PresenceRegistry) Add(userID uint64) (wasOnline bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, wasOnline = p.online[userID]
	p.online[userID]++
	return wasOnline
}

func (p *PresenceRegistry) Remove(userID uint64) (isOffline bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.online[userID] > 1 {
		p.online[userID]--
		return false
	}
	delete(p.online, userID)
	return true
}

func (p *PresenceRegistry) IsOnline(userID uint64) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.online[userID]
	return ok
}
