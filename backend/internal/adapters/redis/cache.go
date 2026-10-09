// Package redis is the go-redis adapter implementing ports.Cache, plus the
// Degrading decorator that makes cache failures non-fatal.
package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache talks to Redis. Redis is optional: Connect never fails hard, it only
// warns, because go-redis reconnects on its own once the server is back.
type Cache struct {
	client  *redis.Client
	timeout time.Duration
}

// Connect parses url, opens the client, and pings once. A failed ping is
// logged, not returned: wrap the result in Degrading and keep going.
func Connect(ctx context.Context, url string, timeout time.Duration) (*Cache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	c := &Cache{client: redis.NewClient(opt), timeout: timeout}
	if err := c.Ping(ctx); err != nil {
		slog.WarnContext(ctx, "redis not reachable at startup; running degraded", slog.Any("err", err))
	}
	return c, nil
}

func (c *Cache) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.timeout)
}

// Ping reports connectivity; used by /readyz.
func (c *Cache) Ping(ctx context.Context) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.client.Ping(ctx).Err()
}

// Close releases the client.
func (c *Cache) Close() error { return c.client.Close() }

// Get implements ports.Cache.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	v, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// GetDel implements ports.Cache.
func (c *Cache) GetDel(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	v, err := c.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// Set implements ports.Cache.
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete implements ports.Cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.client.Del(ctx, key).Err()
}
