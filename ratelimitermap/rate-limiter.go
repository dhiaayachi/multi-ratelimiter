package ratelimitermap

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type limitedEntity interface {
	Key() string
	Kind() string
}

type Limiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type Config struct {
	Rate  rate.Limit
	Burst int
}

type kind struct {
	lock     sync.RWMutex
	limiters map[string]*Limiter
	config   Config
}

type MultiLimiter struct {
	limiters map[string]*kind
	mu       sync.RWMutex
	cancel   context.CancelFunc
	ctx      context.Context
}

func (m *MultiLimiter) Close() {
	m.cancel()
}

func NewMultiLimiter() *MultiLimiter {
	limiters := make(map[string]*kind)
	m := &MultiLimiter{limiters: limiters}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.ctx = ctx
	return m
}

func (m *MultiLimiter) allow(e limitedEntity) bool {
	m.mu.RLock()
	k, exists := m.limiters[e.Kind()]

	if !exists {
		m.mu.RUnlock()
		return true
	}
	m.mu.RUnlock()
	k.lock.RLock()
	l, exists := k.limiters[e.Key()]
	if exists {
		k.lock.RUnlock()
		l.lastAccess = time.Now()
		return l.limiter.Allow()
	}

	// Include the current time when creating a new visitor.

	limiter := rate.NewLimiter(k.config.Rate, k.config.Burst)
	k.lock.RUnlock()
	k.lock.Lock()
	defer k.lock.Unlock()
	l, exists = k.limiters[e.Key()]
	if exists {
		l.lastAccess = time.Now()
		return l.limiter.Allow()
	}
	k.limiters[e.Key()] = &Limiter{limiter, time.Now()}
	return limiter.Allow()
}

func (m *MultiLimiter) Allow(e limitedEntity) bool {
	return m.allow(e)
}

func (m *MultiLimiter) AddKind(k string, c Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters[k] = &kind{limiters: make(map[string]*Limiter), config: c}
	go m.cleanupLimited(m.ctx, k)
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func (m *MultiLimiter) cleanupLimited(ctx context.Context, k string) {
	m.mu.RLock()
	kind, exist := m.limiters[k]
	if !exist {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()
	for {
		waiter := time.After(time.Second)
		//fmt.Println("clean up!!")
		select {
		case <-ctx.Done():
			return
		case <-waiter:
		}
		kind.lock.RLock()
		for ip, v := range kind.limiters {
			if time.Since(v.lastAccess) > 30*time.Millisecond {
				kind.lock.RUnlock()
				kind.lock.Lock()
				delete(kind.limiters, ip)
				kind.lock.Unlock()
				kind.lock.RLock()
			}
		}
		kind.lock.RUnlock()
		//fmt.Println("clean up!!")
	}
}
