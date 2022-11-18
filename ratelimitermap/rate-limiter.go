package ratelimitermap

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type limitedEntity interface {
	Key() string
}

type Limiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type Config struct {
	Rate  rate.Limit
	Burst int
}

type MultiLimiter struct {
	limiters map[string]*Limiter
	mu       sync.RWMutex
	cancel   context.CancelFunc
	ctx      context.Context
	config   Config
}

func (m *MultiLimiter) Close() {
	m.cancel()
}

func NewMultiLimiter(c Config) *MultiLimiter {
	limiters := make(map[string]*Limiter)
	m := &MultiLimiter{limiters: limiters, config: c}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.ctx = ctx
	return m
}

func (m *MultiLimiter) allow(e limitedEntity) bool {
	m.mu.RLock()
	l, exists := m.limiters[e.Key()]
	if exists {
		m.mu.RUnlock()
		l.lastAccess = time.Now()
		return l.limiter.Allow()
	}

	// Include the current time when creating a new visitor.

	limiter := rate.NewLimiter(m.config.Rate, m.config.Burst)
	m.mu.RUnlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	l, exists = m.limiters[e.Key()]
	if exists {
		l.lastAccess = time.Now()
		return l.limiter.Allow()
	}
	m.limiters[e.Key()] = &Limiter{limiter, time.Now()}
	return limiter.Allow()
}

func (m *MultiLimiter) Allow(e limitedEntity) bool {
	return m.allow(e)
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func (m *MultiLimiter) cleanupLimited(ctx context.Context, k string) {
	for {
		waiter := time.After(time.Second)
		select {
		case <-ctx.Done():
			return
		case <-waiter:
		}
		for ip, v := range m.limiters {
			if time.Since(v.lastAccess) > 30*time.Millisecond {
				m.mu.RUnlock()
				m.mu.Lock()
				delete(m.limiters, ip)
				m.mu.Unlock()
				m.mu.RLock()
			}
		}
		m.mu.RUnlock()

	}
}
