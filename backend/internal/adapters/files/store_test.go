package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func key32() []byte { return []byte("0123456789abcdef0123456789abcdef") }

func TestStoreRoundTripEncrypted(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, key32())
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "t1/j1.pdf", []byte("%PDF-1.7 hello")))

	raw, err := os.ReadFile(filepath.Join(dir, "t1", "j1.pdf"))
	require.NoError(t, err)
	require.NotContains(t, string(raw), "hello", "encrypted at rest")

	got, err := s.Get(ctx, "t1/j1.pdf")
	require.NoError(t, err)
	require.Equal(t, "%PDF-1.7 hello", string(got))

	require.NoError(t, s.Delete(ctx, "t1/j1.pdf"))
	require.NoError(t, s.Delete(ctx, "t1/j1.pdf"), "idempotent")
	_, err = s.Get(ctx, "t1/j1.pdf")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStoreRejectsTamperingAndSwaps(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, key32())
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "t1/a.pdf", []byte("A")))
	require.NoError(t, s.Put(ctx, "t2/b.pdf", []byte("B")))

	// A file copied under another key does not decrypt: the key is the AAD.
	raw, err := os.ReadFile(filepath.Join(dir, "t1", "a.pdf"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "t2", "b.pdf"), raw, 0o600))
	_, err = s.Get(ctx, "t2/b.pdf")
	require.Error(t, err)

	raw[len(raw)-1] ^= 1
	require.NoError(t, os.WriteFile(filepath.Join(dir, "t1", "a.pdf"), raw, 0o600))
	_, err = s.Get(ctx, "t1/a.pdf")
	require.Error(t, err)
}

func TestStoreRejectsBadKeys(t *testing.T) {
	_, err := New(t.TempDir(), []byte("short"))
	require.Error(t, err)
	s, err := New(t.TempDir(), key32())
	require.NoError(t, err)
	require.Error(t, s.Put(context.Background(), "../escape.pdf", []byte("x")))
}
