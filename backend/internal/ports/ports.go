// Package ports declares the interfaces the application layer depends on.
// Adapters implement them; use cases consume them.
package ports

import (
	"context"
	"time"
)

// Cache is a byte-oriented key/value cache with TTL. Implementations must
// never make a request fail: see adapters/redis.Degrading.
type Cache interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Pinger reports whether a dependency is reachable. Used by readiness checks.
type Pinger interface {
	Ping(ctx context.Context) error
}
