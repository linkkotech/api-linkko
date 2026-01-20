package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/metric"
)

// RedisRateLimiter implements rate limiting using Redis sliding window algorithm
type RedisRateLimiter struct {
	client              *redis.Client
	rateLimitRejections metric.Int64Counter
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client, rateLimitRejections metric.Int64Counter) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:              client,
		rateLimitRejections: rateLimitRejections,
	}
}

// AllowRequest checks if a request is allowed based on rate limit
// Returns (allowed, remaining, error)
func (rl *RedisRateLimiter) AllowRequest(ctx context.Context, workspaceID string, limit int, windowSeconds int) (bool, int, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(windowSeconds) * time.Second)
	
	key := fmt.Sprintf("ratelimit:workspace:%s", workspaceID)
	
	// Use Redis pipeline for atomic operations
	pipe := rl.client.Pipeline()
	
	// Remove old entries outside the sliding window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
	
	// Add current request timestamp
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	
	// Count requests in current window
	countCmd := pipe.ZCount(ctx, key, "-inf", "+inf")
	
	// Set expiration to twice the window size to ensure cleanup
	pipe.Expire(ctx, key, time.Duration(windowSeconds*2)*time.Second)
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to execute rate limit check: %w", err)
	}
	
	count, err := countCmd.Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to get count: %w", err)
	}
	
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	
	allowed := count <= int64(limit)
	
	// Record rejection metric
	if !allowed && rl.rateLimitRejections != nil {
		rl.rateLimitRejections.Add(ctx, 1)
	}
	
	return allowed, remaining, nil
}
