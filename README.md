# multi-ratelimiter
multi key rate limiter


Benchmark results

```bash
 go test -bench=. -benchtime=60s  -timeout 6000s ./...
?   	github.com/dhiaayachi/rate-limiter-poc	[no test files]
goos: darwin
goarch: amd64
pkg: github.com/dhiaayachi/rate-limiter-poc/ratelimiterGenerics
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkTestRateLimiter_preload-16    	   98095	    795567 ns/op
BenchmarkTestRateLimiter-16            	   79044	    836339 ns/op
PASS
ok  	github.com/dhiaayachi/rate-limiter-poc/ratelimiterGenerics	163.295s
goos: darwin
goarch: amd64
pkg: github.com/dhiaayachi/rate-limiter-poc/ratelimitermap
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkTestRateLimiter_preload-16    	   78627	    859858 ns/op
BenchmarkTestRateLimiter-16            	   87114	    786349 ns/op
PASS
ok  	github.com/dhiaayachi/rate-limiter-poc/ratelimitermap	156.324s
```
