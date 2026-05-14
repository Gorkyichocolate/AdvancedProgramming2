package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// OrderCache handles caching operations for orders
type OrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewOrderCache creates a new order cache instance
func NewOrderCache(redisURL string, ttlSeconds int) (*OrderCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &OrderCache{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}, nil
}

// GetOrderKey generates a cache key for an order
func (oc *OrderCache) GetOrderKey(orderID string) string {
	return fmt.Sprintf("order:%s", orderID)
}

// Get retrieves an order from cache
func (oc *OrderCache) Get(ctx context.Context, orderID string, v any) error {
	key := oc.GetOrderKey(orderID)
	val, err := oc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // Cache miss, not an error
		}
		return fmt.Errorf("redis get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), v); err != nil {
		return fmt.Errorf("unmarshal cache error: %w", err)
	}

	return nil
}

// Set stores an order in cache with TTL
func (oc *OrderCache) Set(ctx context.Context, orderID string, v any) error {
	key := oc.GetOrderKey(orderID)
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := oc.client.Set(ctx, key, data, oc.ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete removes an order from cache
func (oc *OrderCache) Delete(ctx context.Context, orderID string) error {
	key := oc.GetOrderKey(orderID)
	if err := oc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

// InvalidateOrder invalidates an order's cache entry
func (oc *OrderCache) InvalidateOrder(ctx context.Context, orderID string) error {
	return oc.Delete(ctx, orderID)
}

// Close closes the Redis connection
func (oc *OrderCache) Close() error {
	return oc.client.Close()
}
