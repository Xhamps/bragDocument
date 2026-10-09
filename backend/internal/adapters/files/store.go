// Package files is the local-disk FileStore: AES-256-GCM at rest (PRD-0006
// NFR-2), confined to one directory with os.Root.
package files

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Store implements ports.FileStore. The file key is the GCM additional data,
// so a file moved under another key fails to open.
type Store struct {
	root *os.Root
	aead cipher.AEAD
}

// New opens dir (created if missing) with a 32-byte key.
func New(dir string, key []byte) (*Store, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("files: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	return &Store{root: root, aead: aead}, nil
}

// ponytail: whole file in memory; a 2,000-log report is a few MB. Stream (chunked GCM) if reports grow.
func (s *Store) Put(_ context.Context, key string, data []byte) error {
	nonce := make([]byte, s.aead.NonceSize())
	_, _ = rand.Read(nonce)
	sealed := s.aead.Seal(nonce, nonce, data, []byte(key))
	if err := s.root.MkdirAll(path.Dir(key), 0o700); err != nil {
		return fmt.Errorf("files: %w", err)
	}
	tmp := key + ".tmp"
	if err := s.root.WriteFile(tmp, sealed, 0o600); err != nil {
		_ = s.root.Remove(tmp)
		return fmt.Errorf("files: %w", err)
	}
	if err := s.root.Rename(tmp, key); err != nil {
		_ = s.root.Remove(tmp)
		return fmt.Errorf("files: %w", err)
	}
	return nil
}

func (s *Store) Get(_ context.Context, key string) ([]byte, error) {
	sealed, err := s.root.ReadFile(key)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	n := s.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("files: truncated file")
	}
	data, err := s.aead.Open(nil, sealed[:n], sealed[n:], []byte(key))
	if err != nil {
		return nil, fmt.Errorf("files: %s: %w", key, err)
	}
	return data, nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	if err := s.root.Remove(key); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("files: %w", err)
	}
	return nil
}
