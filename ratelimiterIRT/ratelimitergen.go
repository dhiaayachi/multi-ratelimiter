package ratelimiterIRT

import (
	"context"
	radix "github.com/hashicorp/go-immutable-radix"
	"golang.org/x/time/rate"
	"sync/atomic"
	"time"
)

type limitedEntity interface {
	Key() []byte
}

type Limiter struct {
	limiter    *rate.Limiter
	lastAccess atomic.Int64
}

type Config struct {
	Rate         rate.Limit
	Burst        int
	CleanupLimit time.Duration
	CleanupTick  time.Duration
}

type MultiLimiter struct {
	limiters *atomic.Pointer[radix.Tree]
	config   *Config
	cancel   context.CancelFunc
}

func NewMultiLimiter(c Config) *MultiLimiter {
	limiters := &atomic.Pointer[radix.Tree]{}
	limiters.Store(radix.New())
	if c.CleanupLimit == 0 {
		c.CleanupLimit = 30 * time.Millisecond
	}
	if c.CleanupTick == 0 {
		c.CleanupLimit = 1 * time.Second
	}
	m := &MultiLimiter{limiters: limiters, config: &c}
	return m
}

func (m *MultiLimiter) Start() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	m.cancel = cancelFunc
	go func() {
		for {
			m.cleanupLimited(ctx)
		}
	}()
}

func (m *MultiLimiter) Stop() {
	m.cancel()
}

func (m *MultiLimiter) Allow(e limitedEntity) bool {
	limiters := m.limiters.Load()
	l, ok := limiters.Get(e.Key())
	now := time.Now().Unix()
	if ok {
		limiter := l.(*Limiter)
		limiter.lastAccess.Store(now)
		return limiter.limiter.Allow()
	}
	limiter := &Limiter{limiter: rate.NewLimiter(m.config.Rate, m.config.Burst)}
	limiter.lastAccess.Store(now)
	tree, _, _ := limiters.Insert(e.Key(), limiter)
	m.limiters.Store(tree)
	return limiter.limiter.Allow()
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func (m *MultiLimiter) cleanupLimited(ctx context.Context) {
	waiter := time.After(m.config.CleanupTick)

	select {
	case <-ctx.Done():
		return
	case now := <-waiter:
		limiters := m.limiters.Load()
		storedLimiters := limiters
		iter := limiters.Root().Iterator()
		k, v, ok := iter.Next()
		for ok {
			limiter := v.(*Limiter)
			lastAccess := limiter.lastAccess.Load()
			lastAccessT := time.Unix(lastAccess, 0)
			diff := now.Sub(lastAccessT)
			if diff > m.config.CleanupLimit {
				limiters, _, _ = limiters.Delete(k)
			}
			k, v, ok = iter.Next()
		}
		m.limiters.CompareAndSwap(storedLimiters, limiters)
	}
	//

}
