package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var limiters sync.Map

func getIPLimiter(ip string) *rate.Limiter {
	v, _ := limiters.LoadOrStore(ip, &ipLimiter{
		limiter:  rate.NewLimiter(rate.Every(time.Second/10), 20), // 10 req/s, burst 20
		lastSeen: time.Now(),
	})
	il := v.(*ipLimiter)
	il.lastSeen = time.Now()
	return il.limiter
}

func RateLimit() gin.HandlerFunc {
	// Cleanup old entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiters.Range(func(key, value interface{}) bool {
				il := value.(*ipLimiter)
				if time.Since(il.lastSeen) > 10*time.Minute {
					limiters.Delete(key)
				}
				return true
			})
		}
	}()

	return func(c *gin.Context) {
		limiter := getIPLimiter(c.ClientIP())
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "too many requests"},
			})
			return
		}
		c.Next()
	}
}
