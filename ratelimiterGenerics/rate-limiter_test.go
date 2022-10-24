package ratelimitergen

import (
	rate_limiter_poc "github.com/dhiaayachi/rate-limiter-poc"
	"strconv"
	"sync"
	"testing"
)

type ipLimited[K, T comparable] struct {
	key  T
	kind K
}

func (i ipLimited[K, T]) Key() T {
	return i.key
}

func (i ipLimited[K, T]) Kind() K {
	return i.kind
}

func NewIPLimited(n int, p int) *ipLimited[string, string] {
	i := ipLimited[string, string]{key: strconv.Itoa(n), kind: strconv.Itoa(p)}
	return &i
}

func BenchmarkTestMap_preload(b *testing.B) {
	m := NewMultiLimiter[string, string]()

	wg := sync.WaitGroup{}
	ips := make([]*ipLimited[string, string], 0)
	for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
		k := NewIPLimited(i%rate_limiter_poc.NumIPs, i%rate_limiter_poc.NumKind)
		var Config = Config{Rate: 1.0, Burst: 3}
		m.AddKind(k.kind, Config)
		ips = append(ips, k)
		m.Allow(ips[i])
	}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func(ip *ipLimited[string, string]) {
			for j := 0; j < b.N; j++ {
				m.Allow(ip)
			}
			wg.Done()
		}(ips[i])
	}
	wg.Wait()
	m.Close()
}

func BenchmarkTestMap(b *testing.B) {
	m := NewMultiLimiter[string, string]()

	wg := sync.WaitGroup{}
	ips := make([]*ipLimited[string, string], 0)
	p := 0
	for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
		k := NewIPLimited(i%rate_limiter_poc.NumIPs, i%rate_limiter_poc.NumKind)
		var Config = Config{Rate: 1.0, Burst: 3}
		m.AddKind(k.kind, Config)
		ips = append(ips, k)
		p++
		if p > 100 {
			p = 0
		}
	}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func(ip *ipLimited[string, string]) {
			for j := 0; j < b.N; j++ {
				m.Allow(ip)
			}
			wg.Done()
		}(ips[i])
	}
	wg.Wait()
	m.Close()
}
