package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"slices"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Sharing groups the PRD-0004 use cases; one method per file.
type Sharing struct {
	docs   ports.DocumentRepo
	repo   ports.SharingRepo
	mail   ports.Mailer
	appURL string
}

// NewSharing wires the use cases. appURL builds the links in share emails.
func NewSharing(docs ports.DocumentRepo, repo ports.SharingRepo, mail ports.Mailer, appURL string) *Sharing {
	return &Sharing{docs: docs, repo: repo, mail: mail, appURL: appURL}
}

func auditBy(actor domain.User, action, docID, target string, role domain.Role) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actor.ID, ActorEmail: actor.Email, Action: action, DocumentID: docID, Target: target, Role: role}
}

func grantableRole(r domain.Role) error {
	if !r.Grantable() {
		return domain.NewValidationError(map[string]string{"role": "must be editor or viewer"})
	}
	return nil
}

// grantOf returns the user's grant on the document, or domain.ErrNotFound.
func (s *Sharing) grantOf(ctx context.Context, docID, userID string) (domain.Grant, error) {
	sh, err := s.repo.Get(ctx, docID)
	if err != nil {
		return domain.Grant{}, err
	}
	i := slices.IndexFunc(sh.Grants, func(g domain.Grant) bool { return g.UserID == userID })
	if i < 0 {
		return domain.Grant{}, domain.ErrNotFound
	}
	return sh.Grants[i], nil
}

// notify emails the recipient. Failures are logged, never returned (FR-6, ADR-0014).
func (s *Sharing) notify(ctx context.Context, actor domain.User, d domain.Document, to string, role domain.Role, link string) {
	from := cmp.Or(actor.DisplayName, actor.Email)
	subject := fmt.Sprintf("%s shared “%s” with you", from, d.Title)
	body := fmt.Sprintf(`<p>%s shared the brag document <strong>%s</strong> with you as %s.</p>`,
		html.EscapeString(from), html.EscapeString(d.Title), role)
	if s.appURL != "" { // APP_URL unset: a relative href would be useless in an inbox
		body += fmt.Sprintf(`<p><a href="%s">Open it</a></p>`, html.EscapeString(link))
	}
	err := s.mail.Send(ctx, to, subject, body)
	if err != nil && !errors.Is(err, domain.ErrUnavailable) { // unavailable: disabled, logged at startup
		slog.WarnContext(ctx, "share email failed; access was granted anyway",
			slog.String("document_id", d.ID), slog.Any("err", err))
	}
}
