package main

import (
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
)

// ─── Rate Limiting ───────────────────────────────────────────────

type ipLimiter struct {
	limiter *rate.Limiter
}

type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
}

func newRateLimiterStore() *rateLimiterStore {
	return &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
	}
}

func (s *rateLimiterStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.limiters[ip]; !exists {
		s.limiters[ip] = &ipLimiter{
			limiter: rate.NewLimiter(10, 20),
		}
	}

	return s.limiters[ip].limiter
}

var store = newRateLimiterStore()

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests — please slow down",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ─── Auth Middleware ──────────────────────────────────────────────

func (app *application) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			c.Abort()
			return
		}

		claims, err := app.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("userId", claims.UserId)
		c.Next()
	}
}

// ─── Logger Middleware ────────────────────────────────────────────

func loggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
		)
	}
}