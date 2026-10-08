package app

import "github.com/xhamps/bragdocument/backend/internal/ports"

// Documents groups the document use cases; one method per file.
type Documents struct{ docs ports.DocumentRepo }

// NewDocuments wires the use cases.
func NewDocuments(docs ports.DocumentRepo) *Documents { return &Documents{docs: docs} }
