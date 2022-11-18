package ratelimiterIRT

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type Limited struct {
	key string
}

func (l Limited) Key() []byte {
	return []byte(l.key)
}

func TestNewMultiLimiter(t *testing.T) {
	c := Config{Rate: 0.1}
	m := NewMultiLimiter(c)
	require.NotNil(t, m)
	require.NotNil(t, m.limiters)
}

func TestRateLimiterUpdate(t *testing.T) {
	c := Config{Rate: 0.1, CleanupLimit: 1 * time.Millisecond, CleanupTick: 10 * time.Millisecond}
	m := NewMultiLimiter(c)
	m.Allow(Limited{key: "test"})
	limiters := m.limiters.Load()
	l1, ok1 := limiters.Get([]byte("test"))
	require.True(t, ok1)
	require.NotNil(t, l1)
	la1 := l1.(*Limiter).lastAccess.Load()
	m.Allow(Limited{key: "test"})
	limiters = m.limiters.Load()
	l2, ok2 := limiters.Get([]byte("test"))
	require.True(t, ok2)
	require.NotNil(t, l2)
	require.Equal(t, l1, l2)
	la2 := l1.(*Limiter).lastAccess.Load()
	require.Equal(t, la1, la2)

}

func TestRateLimiterCleanup(t *testing.T) {
	c := Config{Rate: 0.1, CleanupLimit: 1 * time.Millisecond, CleanupTick: 10 * time.Millisecond}
	m := NewMultiLimiter(c)
	m.Start()
	m.Allow(Limited{key: "test"})
	limiters := m.limiters.Load()
	l, ok := limiters.Get([]byte("test"))
	require.True(t, ok)
	require.NotNil(t, l)
	time.Sleep(20 * time.Millisecond)
	limiters = m.limiters.Load()
	l, ok = limiters.Get([]byte("test"))
	require.False(t, ok)
	require.Nil(t, l)
	m.Stop()
	m.Allow(Limited{key: "test"})
	time.Sleep(20 * time.Millisecond)
	limiters = m.limiters.Load()
	l, ok = limiters.Get([]byte("test"))
	require.True(t, ok)
	require.NotNil(t, l)
}
