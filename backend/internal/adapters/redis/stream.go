package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Stream publishes outbox messages to Redis Streams and consumes them with a
// consumer group (ADR-0015). One stream per topic: "stream:<topic>".
type Stream struct {
	cache         *Cache
	group, name   string
	claimIdle     time.Duration // pending longer than this is retaken
	maxDeliveries int64         // then dead-lettered
	block         time.Duration // XREADGROUP wait; also bounds how long Consume takes to stop
}

// NewStream shares the cache's client.
func NewStream(c *Cache, group, consumer string) *Stream {
	return &Stream{cache: c, group: group, name: consumer,
		claimIdle: time.Minute, maxDeliveries: 5, block: 5 * time.Second}
}

func key(topic string) string { return "stream:" + topic }

// Publish appends msgs in order; any failure returns an error so the relay
// retries the batch (delivery is at-least-once).
func (s *Stream) Publish(ctx context.Context, msgs []domain.OutboxMessage) error {
	ctx, cancel := s.cache.withTimeout(ctx)
	defer cancel()
	_, err := s.cache.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, m := range msgs {
			p.XAdd(ctx, &redis.XAddArgs{Stream: key(m.Topic), Values: map[string]any{
				"id": m.ID, "tenant_id": m.TenantID, "payload": string(m.Payload)}})
		}
		return nil
	})
	return err
}

// Consume handles topic's messages until ctx is done: new ones via XREADGROUP,
// stale pending ones via XAUTOCLAIM. A handler error leaves the message
// pending; after maxDeliveries it is copied to "<stream>:dead" and acked.
// Redis errors are logged (at most once a minute) and retried, never returned.
func (s *Stream) Consume(ctx context.Context, topic string, handle func(context.Context, domain.OutboxMessage) error) error {
	k := key(topic)
	var ready bool
	var lastWarn time.Time
	for ctx.Err() == nil {
		err := s.poll(ctx, k, topic, &ready, handle)
		if err == nil || ctx.Err() != nil {
			continue
		}
		ready = false // the stream may have been deleted (NOGROUP): recreate the group
		if time.Since(lastWarn) >= time.Minute {
			lastWarn = time.Now()
			slog.WarnContext(ctx, "stream consume failed; retrying", slog.String("stream", k), slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
	return nil
}

// poll runs one round: ensure the group, retake stale pending, read new.
func (s *Stream) poll(ctx context.Context, k, topic string, ready *bool, handle func(context.Context, domain.OutboxMessage) error) error {
	c := s.cache.client
	if !*ready {
		// From "0", not "$": the relay marks rows published right after XADD,
		// so entries written before the group exists must still be delivered.
		err := c.XGroupCreateMkStream(ctx, k, s.group, "0").Err()
		if err != nil && !strings.HasPrefix(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create group: %w", err)
		}
		*ready = true
	}
	claimed, _, err := c.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: k, Group: s.group, Consumer: s.name,
		MinIdle: s.claimIdle, Start: "0-0", Count: 10}).Result()
	if err != nil {
		return fmt.Errorf("autoclaim: %w", err)
	}
	for _, x := range claimed {
		if err := s.deliver(ctx, k, topic, x, handle); err != nil {
			return err
		}
	}
	streams, err := c.XReadGroup(ctx, &redis.XReadGroupArgs{Group: s.group, Consumer: s.name,
		Streams: []string{k, ">"}, Count: 10, Block: s.block}).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("readgroup: %w", err)
	}
	for _, st := range streams {
		for _, x := range st.Messages {
			if err := s.deliver(ctx, k, topic, x, handle); err != nil {
				return err
			}
		}
	}
	return nil
}

// deliver hands one entry to handle and acks it, or dead-letters it once it
// has been delivered maxDeliveries times. Only Redis errors are returned.
func (s *Stream) deliver(ctx context.Context, k, topic string, x redis.XMessage, handle func(context.Context, domain.OutboxMessage) error) error {
	c := s.cache.client
	herr := decodeAndHandle(ctx, topic, x, handle)
	if herr == nil {
		return c.XAck(ctx, k, s.group, x.ID).Err()
	}
	p, err := c.XPendingExt(ctx, &redis.XPendingExtArgs{Stream: k, Group: s.group,
		Start: x.ID, End: x.ID, Count: 1}).Result()
	if err != nil {
		return fmt.Errorf("pending: %w", err)
	}
	if len(p) == 0 || p[0].RetryCount < s.maxDeliveries {
		return nil // stays pending; XAUTOCLAIM retakes it after claimIdle
	}
	_, err = c.TxPipelined(ctx, func(tx redis.Pipeliner) error {
		tx.XAdd(ctx, &redis.XAddArgs{Stream: k + ":dead", Values: x.Values})
		tx.XAck(ctx, k, s.group, x.ID)
		return nil
	})
	if err != nil {
		return fmt.Errorf("dead-letter: %w", err)
	}
	slog.ErrorContext(ctx, "stream message dead-lettered", slog.String("stream", k),
		slog.String("entry", x.ID), slog.Int64("deliveries", p[0].RetryCount), slog.Any("err", herr))
	return nil
}

func decodeAndHandle(ctx context.Context, topic string, x redis.XMessage, handle func(context.Context, domain.OutboxMessage) error) error {
	idStr, _ := x.Values["id"].(string)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("decode id %q: %w", idStr, err)
	}
	tenant, _ := x.Values["tenant_id"].(string)
	payload, _ := x.Values["payload"].(string)
	return handle(ctx, domain.OutboxMessage{ID: id, TenantID: tenant, Topic: topic, Payload: []byte(payload)})
}
