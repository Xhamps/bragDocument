package domain

import (
	"maps"
	"slices"
	"time"
)

// Audit actions (PRD-0009 §7). Stored as text; renaming one needs a migration.
const (
	AuditDocumentCreated    = "document.created"
	AuditDocumentRenamed    = "document.renamed"
	AuditDocumentEdited     = "document.edited"
	AuditDocumentArchived   = "document.archived"
	AuditDocumentUnarchived = "document.unarchived"
	AuditDocumentDeleted    = "document.deleted"

	AuditGrant        = "sharing.granted"
	AuditInvite       = "sharing.invitation_sent"
	AuditRoleChange   = "sharing.role_changed"
	AuditRevoke       = "sharing.revoked"
	AuditInviteCancel = "sharing.invitation_cancelled"
	AuditInviteAccept = "sharing.invitation_accepted"
	AuditTransfer     = "sharing.ownership_transferred"

	AuditLogCreated       = "log.created"
	AuditLogEdited        = "log.edited"
	AuditLogDeleted       = "log.deleted"
	AuditLogStatusChanged = "log.status_changed"

	AuditTelegramLinked   = "telegram.linked"
	AuditTelegramUnlinked = "telegram.unlinked"

	AuditExportRequested = "export.requested"
)

// AuditActions lists every action; the action filter accepts only these.
var AuditActions = []string{
	AuditDocumentCreated, AuditDocumentRenamed, AuditDocumentEdited, AuditDocumentArchived, AuditDocumentUnarchived, AuditDocumentDeleted,
	AuditGrant, AuditInvite, AuditRoleChange, AuditRevoke, AuditInviteCancel, AuditInviteAccept, AuditTransfer,
	AuditLogCreated, AuditLogEdited, AuditLogDeleted, AuditLogStatusChanged,
	AuditTelegramLinked, AuditTelegramUnlinked, AuditExportRequested,
}

// Audit sources (FR-2).
const (
	SourceWeb      = "web"
	SourceTelegram = "telegram"
	SourceSystem   = "system"
)

// Audit target types.
const (
	TargetLog        = "log"
	TargetUser       = "user"
	TargetInvitation = "invitation"
)

// AuditEntry records one action. Writers set ids, action, source, and target;
// the repository copies the actor's name and email and the document title, so
// entries outlive users and documents (FR-12). Never content (FR-13).
type AuditEntry struct {
	ID            int64
	ActorID       string // "" for the system
	ActorName     string
	ActorEmail    string
	Source        string
	Action        string
	DocumentID    string // "" when no document is involved
	DocumentTitle string
	TargetType    string
	TargetID      string
	Target        string // the target's name at the time: log name or email
	Role          Role   // the role granted, changed to, or removed
	ChangedFields []string
	At            time.Time
}

// AuditFilter narrows the audit log. OwnerID is set by the app, never the
// client: it limits entries to documents the owner currently owns (FR-6).
type AuditFilter struct {
	ActorID, DocumentID, Action, OwnerID string
	From, To                             *time.Time // To exclusive
	Before                               int64      // keyset cursor: entries with id < Before
	Limit                                int
}

// AuditPage is one page, newest first. NextBefore is 0 on the last page.
type AuditPage struct {
	Entries    []AuditEntry
	NextBefore int64
}

// AuditActor and AuditDocument feed the filter pickers.
type AuditActor struct{ ID, Name, Email string }

// AuditDocument is a document option in the filter picker.
type AuditDocument struct{ ID, Title string }

// Validate checks the action and date range, then applies the default page
// size and clamps it. It mutates the receiver.
func (f *AuditFilter) Validate() error {
	fields := map[string]string{}
	if f.Action != "" && !slices.Contains(AuditActions, f.Action) {
		fields["action"] = "unknown action"
	}
	if f.From != nil && f.To != nil && !f.To.After(*f.From) {
		fields["to"] = "must be after from"
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	f.Limit = min(f.Limit, 100)
	return nil
}

// DocumentChange names a document update: one entry per action, the biggest
// change wins (archive state > title > description), every field listed.
// "" when nothing changed.
func DocumentChange(old, d Document) (string, []string) {
	var fields []string
	if old.Title != d.Title {
		fields = append(fields, "title")
	}
	if old.Description != d.Description {
		fields = append(fields, "description")
	}
	if old.State != d.State {
		fields = append(fields, "state")
	}
	switch {
	case old.State != d.State && d.State == DocumentArchived:
		return AuditDocumentArchived, fields
	case old.State != d.State:
		return AuditDocumentUnarchived, fields
	case old.Title != d.Title:
		return AuditDocumentRenamed, fields
	case len(fields) > 0:
		return AuditDocumentEdited, fields
	}
	return "", nil
}

// LogChange names a log update by field names only (FR-13): a status-only
// change is status_changed, anything else edited. "" when nothing changed.
func LogChange(old, l Log) (string, []string) {
	changed := map[string]bool{
		"name":        old.Name != l.Name,
		"description": old.Description != l.Description,
		"impact":      old.Impact != l.Impact,
		"status":      old.Status != l.Status,
		"tags":        !slices.Equal(old.Tags, l.Tags),
		"links":       !slices.Equal(old.Links, l.Links),
		"created_at":  !old.CreatedAt.Equal(l.CreatedAt),
	}
	var fields []string
	for _, k := range slices.Sorted(maps.Keys(changed)) {
		if changed[k] {
			fields = append(fields, k)
		}
	}
	switch {
	case len(fields) == 0:
		return "", nil
	case len(fields) == 1 && fields[0] == "status":
		return AuditLogStatusChanged, fields
	}
	return AuditLogEdited, fields
}
