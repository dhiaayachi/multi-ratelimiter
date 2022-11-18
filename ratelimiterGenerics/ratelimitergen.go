package ratelimitergen

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type limitedEntity[T comparable] interface {
	Key() T
}

type Limiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type Config struct {
	Rate  rate.Limit
	Burst int
}

type MultiLimiter[T comparable] struct {
	limiters map[T]*Limiter
	mu       sync.RWMutex
	cancel   context.CancelFunc
	ctx      context.Context
	config   Config
}

func (m *MultiLimiter[T]) Close() {
	m.cancel()
}

func NewMultiLimiter[T comparable](c Config) *MultiLimiter[T] {
	limiters := make(map[T]*Limiter)
	m := &MultiLimiter[T]{limiters: limiters, config: c}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.ctx = ctx
	return m
}

func (m *MultiLimiter[T]) allow(e limitedEntity[T]) bool {
	m.mu.RLock()
	l, exists := m.limiters[e.Key()]
	if exists {
		m.mu.RUnlock()
		return m.allowLocked(l)
	}

	// Include the current time when creating a new visitor.

	limiter := rate.NewLimiter(m.config.Rate, m.config.Burst)

	m.mu.RUnlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	l, exists = m.limiters[e.Key()]
	if exists {
		return m.allowLocked(l)
	}
	m.limiters[e.Key()] = &Limiter{limiter: limiter}
	return m.allowLocked(m.limiters[e.Key()])
}

func (m *MultiLimiter[T]) allowLocked(l *Limiter) bool {
	now := time.Now()
	allow := l.limiter.Allow()
	l.lastAccess = now
	return allow
}

func (m *MultiLimiter[T]) Allow(e limitedEntity[T]) bool {
	return m.allow(e)
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func (m *MultiLimiter[T]) cleanupLimited(ctx context.Context) {

	for {
		waiter := time.After(time.Second)
		//fmt.Println("clean up!!")
		select {
		case <-ctx.Done():
			return
		case <-waiter:
		}
		m.mu.RLock()
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
		//fmt.Println("clean up!!")
	}
}
