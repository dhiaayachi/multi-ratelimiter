# multi-ratelimiter
multi key rate limiter


Benchmark results

```bash
❯ go test -bench=. -benchtime=30s  -timeout 6000s ./...
?   	github.com/dhiaayachi/rate-limiter-poc	[no test files]
goos: darwin
goarch: amd64
pkg: github.com/dhiaayachi/rate-limiter-poc/ratelimiterGenerics
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkTestRateLimiter_preload-16    	    6036	   5833638 ns/op
BenchmarkTestRateLimiter-16            	    4442	   7233422 ns/op
PASS
ok  	github.com/dhiaayachi/rate-limiter-poc/ratelimiterGenerics	69.500s
goos: darwin
goarch: amd64
pkg: github.com/dhiaayachi/rate-limiter-poc/ratelimiterIRT
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkTestRateLimiter_preload-16    	    8671	   3672943 ns/op
BenchmarkTestRateLimiter-16            	    1131	  48952808 ns/op
PASS
ok  	github.com/dhiaayachi/rate-limiter-poc/ratelimiterIRT	91.661s
goos: darwin
goarch: amd64
pkg: github.com/dhiaayachi/rate-limiter-poc/ratelimitermap
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkTestRateLimiter_preload-16    	    4562	   6858848 ns/op
BenchmarkTestRateLimiter-16            	    4042	   7861634 ns/op
PASS
ok  	github.com/dhiaayachi/rate-limiter-poc/ratelimitermap	65.492s
```
