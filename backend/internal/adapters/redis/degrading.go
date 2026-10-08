package redis

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/xhamps/bragdocument/backend/internal/ports"
)

const degradedLogInterval = time.Minute

// Degrading wraps a ports.Cache so that every error becomes a cache miss.
// It counts errors in cache_errors_total and logs at most once per minute.
// cmd always wires this around the real cache: a feature may fail because
// Postgres is down, never because Redis is.
type Degrading struct {
	inner   ports.Cache
	errors  prometheus.Counter
	healthy atomic.Bool
	lastLog atomic.Int64
}

// NewDegrading registers the error counter on reg and returns the decorator.
func NewDegrading(inner ports.Cache, reg prometheus.Registerer) *Degrading {
	d := &Degrading{
		inner: inner,
		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cache_errors_total",
			Help: "Cache operations that failed and were treated as misses.",
		}),
	}
	d.healthy.Store(true)
	reg.MustRegister(d.errors)
	return d
}

// Healthy is false after the last operation failed, true after one succeeds.
func (d *Degrading) Healthy() bool { return d.healthy.Load() }

func (d *Degrading) fail(ctx context.Context, op string, err error) {
	if errors.Is(err, context.Canceled) {
		return // client went away; a miss, not a cache outage
	}
	d.errors.Inc()
	d.healthy.Store(false)
	now := time.Now().Unix()
	last := d.lastLog.Load()
	if now-last >= int64(degradedLogInterval.Seconds()) && d.lastLog.CompareAndSwap(last, now) {
		slog.WarnContext(ctx, "cache degraded, treating as miss", slog.String("op", op), slog.Any("err", err))
	}
}

// Get implements ports.Cache; errors become misses.
func (d *Degrading) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, found, err := d.inner.Get(ctx, key)
	if err != nil {
		d.fail(ctx, "get", err)
		return nil, false, nil
	}
	d.healthy.Store(true)
	return v, found, nil
}

// Set implements ports.Cache; errors are dropped.
func (d *Degrading) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := d.inner.Set(ctx, key, value, ttl); err != nil {
		d.fail(ctx, "set", err)
		return nil
	}
	d.healthy.Store(true)
	return nil
}

// Delete implements ports.Cache; errors are dropped.
func (d *Degrading) Delete(ctx context.Context, key string) error {
	if err := d.inner.Delete(ctx, key); err != nil {
		d.fail(ctx, "delete", err)
		return nil
	}
	d.healthy.Store(true)
	return nil
}
