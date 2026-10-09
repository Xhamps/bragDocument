//go:build integration

package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func startRedis(t *testing.T) (testcontainers.Container, string) {
	t.Helper()
	ctx := context.Background()
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })
	ep, err := ctr.PortEndpoint(ctx, "6379/tcp", "redis")
	require.NoError(t, err)
	return ctr, ep
}

// recorder counts deliveries per outbox id; failing ids return an error.
type recorder struct {
	mu   sync.Mutex
	seen map[int64]int
	fail map[int64]bool
}

func (r *recorder) handle(_ context.Context, m domain.OutboxMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen[m.ID]++
	if r.fail[m.ID] {
		return fmt.Errorf("boom %d", m.ID)
	}
	return nil
}

func (r *recorder) count(id int64) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[id]
}

func TestStreamPublishConsume(t *testing.T) {
	ctr, url := startRedis(t)
	ctx := context.Background()
	cache, err := Connect(ctx, url, 2*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cache.Close() })

	s := NewStream(cache, "bragdoc", "test-consumer")
	s.claimIdle, s.maxDeliveries, s.block = 100*time.Millisecond, 3, 100*time.Millisecond
	key := "stream:" + domain.TopicAudit

	msg := func(id int64) domain.OutboxMessage {
		return domain.OutboxMessage{ID: id, TenantID: "00000000-0000-0000-0000-00000000000a",
			Topic: domain.TopicAudit, Payload: []byte(fmt.Sprintf(`{"n":%d}`, id))}
	}

	// 0. Published to a brand-new stream before any group exists: still delivered.
	require.NoError(t, s.Publish(ctx, []domain.OutboxMessage{msg(1)}))

	rec := &recorder{seen: map[int64]int{}, fail: map[int64]bool{3: true}}
	cctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- s.Consume(cctx, domain.TopicAudit, rec.handle) }()
	require.Eventually(t, func() bool { return rec.count(1) == 1 }, 5*time.Second, 20*time.Millisecond)

	// 1. A message published while consuming is handled once; all are acked.
	require.NoError(t, s.Publish(ctx, []domain.OutboxMessage{msg(2)}))
	require.Eventually(t, func() bool { return rec.count(2) == 1 }, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		p, err := cache.client.XPending(ctx, key, "bragdoc").Result()
		return err == nil && p.Count == 0
	}, 5*time.Second, 20*time.Millisecond)

	// 2. A failing message is redelivered, then dead-lettered and acked.
	require.NoError(t, s.Publish(ctx, []domain.OutboxMessage{msg(3)}))
	dead := key + ":dead"
	require.Eventually(t, func() bool {
		n, err := cache.client.XLen(ctx, dead).Result()
		return err == nil && n == 1
	}, 10*time.Second, 50*time.Millisecond)
	require.Equal(t, 3, rec.count(3), "delivered maxDeliveries times")
	p, err := cache.client.XPending(ctx, key, "bragdoc").Result()
	require.NoError(t, err)
	require.Zero(t, p.Count)
	entries, err := cache.client.XRange(ctx, dead, "-", "+").Result()
	require.NoError(t, err)
	require.Equal(t, map[string]any{"id": "3", "tenant_id": "00000000-0000-0000-0000-00000000000a",
		"payload": `{"n":3}`}, entries[0].Values)
	require.Equal(t, 1, rec.count(1), "acked messages are not redelivered")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Consume did not return after cancel")
	}

	// 3. Publish fails while Redis is down, so the relay retries.
	require.NoError(t, ctr.Stop(ctx, nil))
	require.Error(t, s.Publish(ctx, []domain.OutboxMessage{msg(4)}))
}
