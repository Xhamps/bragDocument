package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// errNoChange from a write's fn commits the transaction without an audit
// message: the call was a no-op (e.g. unlinking an account that was not linked).
var errNoChange = errors.New("no change")

// inTenantTx runs fn in a transaction scoped to the context's tenant and returns its result.
func inTenantTx[T any](ctx context.Context, db *DB, fn func(ctx context.Context, q *sqlcgen.Queries) (T, error)) (T, error) {
	var out T
	err := withQueries(ctx, db, func(ctx context.Context, q *sqlcgen.Queries) error {
		var err error
		out, err = fn(ctx, q)
		return err
	})
	return out, err
}

// enqueue writes one outbox message in the caller's transaction (ADR-0015).
func enqueue(ctx context.Context, q *sqlcgen.Queries, topic string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return wrap(q.EnqueueOutbox(ctx, sqlcgen.EnqueueOutboxParams{Topic: topic, Payload: b}))
}

// write runs a resource's insert, update, or delete and its audit message in
// one tenant transaction, and returns fn's result. Every repo write goes
// through it (PRD-0009 FR-1). The actor and document are snapshotted before
// fn when a.DocumentID is known, while the document still exists (FR-12); the
// message is enqueued after fn, once every id is known. fn may fill ids the
// database assigns.
func write[T any](ctx context.Context, db *DB, a domain.AuditEntry,
	fn func(ctx context.Context, q *sqlcgen.Queries, a *domain.AuditEntry) (T, error)) (T, error) {
	return inTenantTx(ctx, db, func(ctx context.Context, q *sqlcgen.Queries) (T, error) {
		var zero T
		snapped := a.DocumentID != ""
		if snapped {
			if err := snapshot(ctx, q, &a); err != nil {
				return zero, err
			}
		}
		out, err := fn(ctx, q, &a)
		if errors.Is(err, errNoChange) {
			return out, nil
		}
		if err != nil {
			return zero, err
		}
		if !snapped {
			if err := snapshot(ctx, q, &a); err != nil {
				return zero, err
			}
		}
		return out, enqueueAudit(ctx, q, a)
	})
}

// auditMessage is the audit.entry payload; AuditRepo.Store decodes the same struct.
type auditMessage struct {
	ActorID       string    `json:"actor_id"`
	ActorName     string    `json:"actor_name"`
	ActorEmail    string    `json:"actor_email"`
	Source        string    `json:"source"`
	Action        string    `json:"action"`
	DocumentID    string    `json:"document_id"`
	DocumentTitle string    `json:"document_title"`
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	Target        string    `json:"target"`
	Role          string    `json:"role"`
	ChangedFields []string  `json:"changed_fields"`
	At            time.Time `json:"at"`
}

// snapshot copies the actor's name and email and the document title into a.
// An actor or document id that matches nothing is ErrNotFound.
func snapshot(ctx context.Context, q *sqlcgen.Queries, a *domain.AuditEntry) error {
	aid, err := optID(a.ActorID)
	if err != nil {
		return err
	}
	did, err := optID(a.DocumentID)
	if err != nil {
		return err
	}
	s, err := q.AuditSnapshot(ctx, sqlcgen.AuditSnapshotParams{ActorID: aid, DocumentID: did})
	if err != nil {
		return wrap(err)
	}
	a.ActorName, a.ActorEmail, a.DocumentTitle = s.ActorName, s.ActorEmail, s.DocumentTitle.String
	return nil
}

// enqueueAudit enqueues a, already snapshotted, as an audit.entry message.
// ponytail: at is the app clock at enqueue, and entries are listed in consume
// order (ListAudit by id); order by (at, id) if strict ordering matters.
func enqueueAudit(ctx context.Context, q *sqlcgen.Queries, a domain.AuditEntry) error {
	return enqueue(ctx, q, domain.TopicAudit, auditMessage{
		ActorID: a.ActorID, ActorName: a.ActorName, ActorEmail: a.ActorEmail, Source: a.Source, Action: a.Action,
		DocumentID: a.DocumentID, DocumentTitle: a.DocumentTitle, TargetType: a.TargetType, TargetID: a.TargetID,
		Target: a.Target, Role: string(a.Role), ChangedFields: orEmpty(a.ChangedFields), At: time.Now().UTC(),
	})
}
