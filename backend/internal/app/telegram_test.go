package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type tgFixture struct {
	tg    *Telegram
	lf    logsFixture
	links *fakeTelegramLinks
	codes *fakeCache
	undo  *fakeCache
	scope string // last tenant passed to the scope func
}

func newTGFixture() *tgFixture {
	f := &tgFixture{lf: newLogsFixture(), links: newFakeTelegramLinks(), codes: newFakeCache(), undo: newFakeCache()}
	f.tg = NewTelegram(f.links, f.lf.docs, f.lf.s, f.codes, f.undo, "https://app.test",
		func(ctx context.Context, tenantID string) context.Context { f.scope = tenantID; return ctx })
	f.tg.now = f.lf.s.now
	f.tg.newCode = func() string { return "ABCD2345" }
	return f
}

// linked links Telegram user 42 to u1 (tenant t1) with target doc.
func (f *tgFixture) linked(doc string) {
	f.links.byUser["u1"] = domain.TelegramLink{UserID: "u1", TenantID: "t1", TelegramUserID: 42, DocumentID: doc}
}

func TestTelegramNewCode(t *testing.T) {
	f := newTGFixture()
	c, err := f.tg.NewCode(context.Background(), "u1", "t1")
	require.NoError(t, err)
	require.Equal(t, "ABCD2345", c.Code)
	require.Equal(t, f.tg.now().Add(10*time.Minute), c.ExpiresAt)
	require.Equal(t, 10*time.Minute, f.codes.ttl["tg:code:ABCD2345"])

	f.codes.err = errors.New("redis down")
	_, err = f.tg.NewCode(context.Background(), "u1", "t1")
	require.ErrorIs(t, err, domain.ErrUnavailable)
}

func TestTelegramRandomCode(t *testing.T) {
	c := randomCode()
	require.Len(t, c, linkCodeLen)
	for _, r := range c {
		require.Contains(t, linkCodeAlphabet, string(r))
	}
}

func TestTelegramStartLinks(t *testing.T) {
	f := newTGFixture()
	_, err := f.tg.NewCode(context.Background(), "u1", "t1")
	require.NoError(t, err)

	require.Contains(t, f.tg.Reply(context.Background(), 42, "/start abcd2345"), "Linked")
	l, err := f.links.Get(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, int64(42), l.TelegramUserID)
	require.Equal(t, "t1", l.TenantID)

	// Single use.
	require.Contains(t, f.tg.Reply(context.Background(), 43, "/start ABCD2345"), "invalid or expired")
}

func TestTelegramStartFailures(t *testing.T) {
	f := newTGFixture()
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/start NOPE"), "invalid or expired")

	f.links.byUser["u2"] = domain.TelegramLink{UserID: "u2", TenantID: "t1", TelegramUserID: 42}
	_, _ = f.tg.NewCode(context.Background(), "u1", "t1")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/start ABCD2345"), "already linked to another account")

	_, _ = f.tg.NewCode(context.Background(), "u1", "t1")
	f.links.err = errors.New("db down")
	require.Equal(t, msgTryAgain, f.tg.Reply(context.Background(), 7, "/start ABCD2345"))

	f.codes.err = errors.New("redis down")
	require.Contains(t, f.tg.Reply(context.Background(), 7, "/start ABCD2345"), "temporarily unavailable")
}

func TestTelegramStatusAndUnlink(t *testing.T) {
	f := newTGFixture()
	_, ok, err := f.tg.Status(context.Background(), "u1")
	require.NoError(t, err)
	require.False(t, ok)
	f.linked("d1")
	l, ok, err := f.tg.Status(context.Background(), "u1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "d1", l.DocumentID)
	require.NoError(t, f.tg.Unlink(context.Background(), "u1"))
	require.NoError(t, f.tg.Unlink(context.Background(), "u1"))
	require.Contains(t, f.tg.Reply(context.Background(), 42, "hello"), "/start")
}
