package domain

import "time"

// Grant gives a tenant member a role on a document (PRD-0004 §10).
type Grant struct {
	DocumentID  string
	UserID      string
	Email       string
	DisplayName string
	Role        Role
	GrantedBy   string
	GrantedAt   time.Time
}

// DocumentInvitation holds a role for an email that is not a user yet (FR-4).
// The first sign-in with that email joins the tenant and turns it into a Grant.
type DocumentInvitation struct {
	ID         string
	DocumentID string
	Email      string
	Role       Role
	InvitedBy  string
	CreatedAt  time.Time
}

// Sharing is what the owner sees in the share panel.
type Sharing struct {
	Grants      []Grant
	Invitations []DocumentInvitation // pending only
}

// Audit actions (FR-8).
const (
	AuditGrant        = "grant"
	AuditInvite       = "invite"
	AuditRoleChange   = "role_change"
	AuditRevoke       = "revoke"
	AuditInviteCancel = "invite_cancel"
	AuditInviteAccept = "invite_accept"
	AuditTransfer     = "transfer"
)

// AuditEntry records one sharing change. The repository copies the document
// title; emails are copied by the caller, so entries outlive users and documents.
type AuditEntry struct {
	ID            int64
	ActorID       string
	ActorEmail    string
	Action        string
	DocumentID    string
	DocumentTitle string
	Target        string // email of the user or invitee acted on
	Role          Role   // the role granted, changed to, or removed
	At            time.Time
}
