package ratelimitergen

import (
	"context"
	"fmt"
	"github.com/armon/go-metrics"
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
	count      float32
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
	countCh  chan float32
	metrics  *metrics.Metrics
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
	inm := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics, err := metrics.NewGlobal(metrics.DefaultConfig("service-name"), inm)
	if err != nil {
		return nil
	}
	m.countCh = make(chan float32, 32)
	m.metrics = metrics
	go m.metricsRoutine(ctx)
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
		return m.allowLocked(l)
	}

	// Include the current time when creating a new visitor.

	limiter := rate.NewLimiter(k.config.Rate, k.config.Burst)

	k.lock.RUnlock()
	k.lock.Lock()
	defer k.lock.Unlock()
	l, exists = k.limiters[e.Key()]
	if exists {
		return m.allowLocked(l)
	}
	k.limiters[e.Key()] = &Limiter{limiter: limiter}
	return m.allowLocked(k.limiters[e.Key()])
}

func (m *MultiLimiter[K, T]) allowLocked(l *Limiter) bool {
	now := time.Now()
	allow := l.limiter.Allow()
	if !allow {
		l.count++
		if now.Sub(l.lastAccess) > 1*time.Second {
			l.count = 0
		}
		select {
		case m.countCh <- l.count:
		default:
		}
	}
	l.lastAccess = now
	return allow
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

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func (m *MultiLimiter[K, T]) metricsRoutine(ctx context.Context) {
	for {
		var max float32
		select {
		case v := <-m.countCh:
			if v > max {
				fmt.Printf("max:%f\n", v)
				m.metrics.AddSample([]string{"max_clients"}, v)
			}
		case <-ctx.Done():
			return
		}
	}
}
