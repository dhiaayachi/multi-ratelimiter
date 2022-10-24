package ratelimitermap

import (
	rate_limiter_poc "github.com/dhiaayachi/rate-limiter-poc"
	"strconv"
	"sync"
	"testing"
)

type ipLimited struct {
	key  string
	kind string
}

func (i ipLimited) Key() string {
	return i.key
}

func (i ipLimited) Kind() string {
	return i.kind
}

func NewIPLimited(n int, p int) *ipLimited {
	i := ipLimited{key: strconv.Itoa(n), kind: strconv.Itoa(p)}
	return &i
}

func BenchmarkTestMap_preload(b *testing.B) {
	m := NewMultiLimiter()

	wg := sync.WaitGroup{}
	ips := make([]*ipLimited, 0)
	for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
		k := NewIPLimited(i%rate_limiter_poc.NumIPs, i%rate_limiter_poc.NumKind)
		var Config = Config{Rate: 1.0, Burst: 3}
		m.AddKind(k.kind, Config)
		ips = append(ips, k)
		m.Allow(ips[i])

	}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func(ip *ipLimited) {
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
	m := NewMultiLimiter()

	wg := sync.WaitGroup{}
	ips := make([]*ipLimited, 0)
	for i := 0; i < rate_limiter_poc.NumKind*rate_limiter_poc.NumIPs; i++ {
		k := NewIPLimited(i%rate_limiter_poc.NumIPs, i%rate_limiter_poc.NumKind)
		var Config = Config{Rate: 1.0, Burst: 3}
		m.AddKind(k.kind, Config)
		ips = append(ips, k)
	}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func(ip *ipLimited) {
			for j := 0; j < b.N; j++ {
				m.Allow(ip)
			}
			wg.Done()
		}(ips[i])
	}
	wg.Wait()
	m.Close()
}
