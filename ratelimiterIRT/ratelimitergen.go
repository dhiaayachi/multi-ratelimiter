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
	Rate  rate.Limit
	Burst int
}

type MultiLimiter struct {
	limiters *atomic.Pointer[radix.Tree]
	config   *Config
	cancel   context.CancelFunc
}

func NewMultiLimiter(c Config) *MultiLimiter {
	limiters := &atomic.Pointer[radix.Tree]{}
	limiters.Store(radix.New())
	m := &MultiLimiter{limiters: limiters, config: &c}
	ctx, cancelFunc := context.WithCancel(context.Background())
	m.cancel = cancelFunc
	go m.cleanupLimited(ctx)
	return m
}

func (m *MultiLimiter) Close() {
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
	for {
		waiter := time.NewTicker(time.Second)
		//fmt.Println("clean up!!")
		select {
		case <-ctx.Done():
			return
		case now := <-waiter.C:
			limiters := m.limiters.Load()
			storedLimiters := limiters
			iter := limiters.Root().Iterator()
			k, v, ok := iter.Next()
			for ok {
				limiter := v.(*Limiter)
				if limiter.lastAccess.Load() < now.Add(-30*time.Millisecond).Unix() {
					limiters, _, _ = limiters.Delete(k)
					//fmt.Println("clean up!!")
				}
				k, v, ok = iter.Next()
			}
			m.limiters.CompareAndSwap(storedLimiters, limiters)
		}
		//
	}
}
