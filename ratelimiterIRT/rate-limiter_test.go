package ratelimiterIRT

import (
	rate_limiter_poc "github.com/dhiaayachi/rate-limiter-poc"
	"strconv"
	"sync"
	"testing"
)

type ipLimited struct {
	key []byte
}

func (i ipLimited) Key() []byte {
	return i.key
}

func NewIPLimited(n int, p int) *ipLimited {
	i := ipLimited{key: []byte(strconv.Itoa(n))}
	return &i
}

func BenchmarkTestRateLimiter_preload(b *testing.B) {
	var Config = Config{Rate: 1.0, Burst: 500}
	m := NewMultiLimiter(Config)
	m.Start()

	wg := sync.WaitGroup{}
	ips := make([]*ipLimited, 0)
	for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
		k := NewIPLimited(i%rate_limiter_poc.NumIPs, i%rate_limiter_poc.NumKind)
		ips = append(ips, k)
		m.Allow(ips[i])
	}
	for j := 0; j < b.N; j++ {
		wg.Add(1)
		go func() {
			for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
				m.Allow(ips[i])
			}
			wg.Done()
		}()
	}
	wg.Wait()
	m.Stop()
}

func BenchmarkTestRateLimiter(b *testing.B) {
	var Config = Config{Rate: 1.0, Burst: 500}
	m := NewMultiLimiter(Config)

	wg := sync.WaitGroup{}
	ips := make([]*ipLimited, 0)
	for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
		k := NewIPLimited(i%rate_limiter_poc.NumIPs, i%rate_limiter_poc.NumKind)
		ips = append(ips, k)
	}
	for j := 0; j < b.N; j++ {
		wg.Add(1)
		go func() {
			for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
				m.Allow(ips[i])
			}
			wg.Done()
		}()
	}
	wg.Wait()
	m.Stop()
}
