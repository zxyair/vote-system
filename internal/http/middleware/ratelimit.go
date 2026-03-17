package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type userLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// TokenBucket provides a global limiter + per-user limiter (by X-User-Id).
func TokenBucket(globalRPS float64, globalBurst int, userRPS float64, userBurst int) gin.HandlerFunc {
	global := rate.NewLimiter(rate.Limit(globalRPS), globalBurst)

	var (
		mu      sync.Mutex
		byUser  = map[string]*userLimiter{}
		ttl     = 10 * time.Minute
		cleaned = time.Now()
	)

	return func(c *gin.Context) {
		if !global.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
			return
		}

		uid := c.GetHeader(HeaderUserID)
		if uid != "" {
			mu.Lock()
			ul := byUser[uid]
			if ul == nil {
				ul = &userLimiter{limiter: rate.NewLimiter(rate.Limit(userRPS), userBurst)}
				byUser[uid] = ul
			}
			ul.lastSeen = time.Now()

			// opportunistic cleanup
			if time.Since(cleaned) > ttl {
				now := time.Now()
				for k, v := range byUser {
					if now.Sub(v.lastSeen) > ttl {
						delete(byUser, k)
					}
				}
				cleaned = now
			}
			ok := ul.limiter.Allow()
			mu.Unlock()

			if !ok {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "user rate limited"})
				return
			}
		}

		c.Next()
	}
}
