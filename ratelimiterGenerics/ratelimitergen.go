package ratelimitergen

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type limitedEntity[K, T comparable] interface {
	Key() T
	Kind() K
}

type Limiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type Config struct {
	Rate  rate.Limit
	Burst int
}

type kind[T comparable] struct {
	lock     sync.RWMutex
	limiters map[T]*Limiter
	config   Config
}

type MultiLimiter[K, T comparable] struct {
	limiters map[K]*kind[T]
	mu       sync.RWMutex
	cancel   context.CancelFunc
	ctx      context.Context
}

func (m *MultiLimiter[K, T]) Close() {
	m.cancel()
}

func NewMultiLimiter[K, T comparable]() *MultiLimiter[K, T] {
	limiters := make(map[K]*kind[T])
	m := &MultiLimiter[K, T]{limiters: limiters}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.ctx = ctx
	return m
}

func (m *MultiLimiter[K, T]) allow(e limitedEntity[K, T]) bool {
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

func (m *MultiLimiter[K, T]) Allow(e limitedEntity[K, T]) bool {
	return m.allow(e)
}

func (m *MultiLimiter[K, T]) AddKind(k K, c Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters[k] = &kind[T]{limiters: make(map[T]*Limiter), config: c}

	go m.cleanupLimited(m.ctx, k)
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func (m *MultiLimiter[K, T]) cleanupLimited(ctx context.Context, k K) {
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
