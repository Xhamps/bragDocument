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

// titled titles the logs fixture's documents: d1 and d3 active, d2 archived.
func (f *tgFixture) titled() {
	for id, title := range map[string]string{"d1": "Alpha", "d2": "Beta", "d3": "Gamma"} {
		d := f.lf.docs.docs[id]
		d.Title = title
		f.lf.docs.docs[id] = d
	}
}

func TestTelegramUnlinkedStoresNothing(t *testing.T) {
	f := newTGFixture()
	for _, msg := range []string{"Shipped X", "/docs", "/use 1", "/last", "/undo", "/help"} {
		require.Equal(t, msgNotLinked, f.tg.Reply(context.Background(), 42, msg), msg)
	}
	require.Empty(t, f.lf.logs.logs)
	require.Empty(t, f.undo.data)
	require.Empty(t, f.scope, "no tenant resolved")
}

func TestTelegramHelp(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	require.Equal(t, msgHelp, f.tg.Reply(context.Background(), 42, "/help"))
	require.Equal(t, msgHelp, f.tg.Reply(context.Background(), 42, "/start"))
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/nope"), "/help")
}

func TestTelegramDocsAndUse(t *testing.T) {
	f := newTGFixture()
	f.titled()
	f.linked("d3")
	out := f.tg.Reply(context.Background(), 42, "/docs")
	require.Equal(t, "t1", f.scope)
	require.Contains(t, out, "1. Alpha\n2. Gamma ✓\n")
	require.NotContains(t, out, "Beta", "archived excluded")

	for _, bad := range []string{"/use", "/use 0", "/use 3", "/use x"} {
		require.Contains(t, f.tg.Reply(context.Background(), 42, bad), "/docs", bad)
	}
	require.Equal(t, "d3", f.links.byUser["u1"].DocumentID, "unchanged")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/use@BragBot 1"), "Now writing to “Alpha”")
	require.Equal(t, "d1", f.links.byUser["u1"].DocumentID)

	f.lf.docs.docs = map[string]domain.Document{}
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/docs"), "no active documents")
}

func TestTelegramMessageCreatesLog(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	out := f.tg.Reply(context.Background(), 42, "Shipped SSO #auth !high https://github.com/x/pr/1\nCut tickets by 40%")
	require.Len(t, f.lf.logs.logs, 1)
	l := f.lf.logs.logs["l1"]
	require.Equal(t, "Shipped SSO", l.Name)
	require.Equal(t, "Cut tickets by 40%", l.Description)
	require.Equal(t, "high", l.Impact)
	require.Equal(t, []string{"auth"}, l.Tags)
	require.Equal(t, "https://github.com/x/pr/1", l.Links[0].URL)
	require.Equal(t, domain.StatusDone, l.Status)
	require.Equal(t, "u1", l.CreatedBy)
	require.Equal(t, "t1", l.TenantID)
	require.Equal(t, "t1", f.scope)
	require.Equal(t, "Logged: Shipped SSO\nImpact: high\nTags: #auth\nLink: https://github.com/x/pr/1"+
		"\nEdit: https://app.test/documents/d1?edit=l1\n/undo to remove it.", out)
	require.Equal(t, undoTTL, f.undo.ttl["tg:undo:42"])
}

func TestTelegramNoImpactWarning(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	require.Contains(t, f.tg.Reply(context.Background(), 42, "Did a thing"), "No impact stated")

	f.lf.impact.err = domain.ErrUnavailable // extraction off: nil statement, no nagging
	require.NotContains(t, f.tg.Reply(context.Background(), 42, "Did a thing"), "No impact stated")
}

func TestTelegramMessageErrors(t *testing.T) {
	f := newTGFixture()
	f.linked("")
	require.Equal(t, msgPickDoc, f.tg.Reply(context.Background(), 42, "X"))
	f.linked("d2") // archived
	require.Equal(t, msgArchived, f.tg.Reply(context.Background(), 42, "X"))
	f.lf.docs.docs["d9"] = domain.Document{ID: "d9", TenantID: "t1", OwnerID: "u9", State: domain.DocumentActive}
	f.linked("d9") // someone else's
	require.Equal(t, msgNoAccess, f.tg.Reply(context.Background(), 42, "X"))
	f.linked("gone")
	require.Equal(t, msgNoAccess, f.tg.Reply(context.Background(), 42, "X"))
	f.linked("d1")
	require.Equal(t, "Not saved. name: first line needs some text besides tags and links",
		f.tg.Reply(context.Background(), 42, "#only"))
	require.Contains(t, f.tg.Reply(context.Background(), 42, "X !huge"), "Logged", "unknown impact token stays in the name")
	require.Equal(t, msgTryAgain, f.tg.failed(context.Background(), errors.New("boom")))
	require.Len(t, f.lf.logs.logs, 1)
}

func TestTelegramLastAndUndo(t *testing.T) {
	f := newTGFixture()
	f.linked("")
	require.Equal(t, msgPickDoc, f.tg.Reply(context.Background(), 42, "/last"))
	f.linked("d1")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/last"), "No logs yet")
	require.Equal(t, msgNothingToUndo, f.tg.Reply(context.Background(), 42, "/undo"))
	f.tg.Reply(context.Background(), 42, "First")
	f.tg.Reply(context.Background(), 42, "Second")
	require.Equal(t, "• Second (medium, Oct 8)\n• First (medium, Oct 8)", f.tg.Reply(context.Background(), 42, "/last"))
	require.Equal(t, 5, f.lf.logs.filter.PerPage)

	require.Equal(t, "Removed: Second", f.tg.Reply(context.Background(), 42, "/undo"))
	require.Len(t, f.lf.logs.logs, 1)
	require.Equal(t, msgNothingToUndo, f.tg.Reply(context.Background(), 42, "/undo"), "only the last one")

	// Already deleted on the web: the pointer is still cleared.
	f.tg.Reply(context.Background(), 42, "Third")
	delete(f.lf.logs.logs, "l3")
	require.Equal(t, "Removed: Third", f.tg.Reply(context.Background(), 42, "/undo"))

	// Archived since: refused, pointer kept.
	f.tg.Reply(context.Background(), 42, "Fourth")
	d := f.lf.docs.docs["d1"]
	d.State = domain.DocumentArchived
	f.lf.docs.docs["d1"] = d
	require.Equal(t, msgArchived, f.tg.Reply(context.Background(), 42, "/undo"))
	require.Contains(t, f.undo.data, "tg:undo:42")
}
