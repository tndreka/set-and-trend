package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	il := &ipLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
	// Cleanup stale entries every 5 minutes.
	go func() {
		for range time.Tick(5 * time.Minute) {
			il.mu.Lock()
			il.limiters = make(map[string]*rate.Limiter)
			il.mu.Unlock()
		}
	}()
	return il
}

func (il *ipLimiter) get(ip string) *rate.Limiter {
	il.mu.Lock()
	defer il.mu.Unlock()
	l, ok := il.limiters[ip]
	if !ok {
		l = rate.NewLimiter(il.rate, il.burst)
		il.limiters[ip] = l
	}
	return l
}

// RateLimitMiddleware returns a Gin middleware that limits requests per IP.
// rps is requests per second, burst is the max burst size.
func RateLimitMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := newIPLimiter(rate.Limit(rps), burst)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.get(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			c.Abort()
			return
		}
		c.Next()
	}
}
