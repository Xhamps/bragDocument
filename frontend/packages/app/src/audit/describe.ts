import type { AuditAction, AuditEntry } from "../lib/types";

const t = (a: AuditEntry) => `“${a.target.name}”`;
const fields = (a: AuditEntry) =>
  a.changed_fields.length ? ` (${a.changed_fields.join(", ")})` : "";

const verbs: Record<AuditAction, (a: AuditEntry) => string> = {
  "document.created": () => "created the document",
  "document.renamed": (a) => `renamed the document${fields(a)}`,
  "document.edited": (a) => `edited the document${fields(a)}`,
  "document.archived": () => "archived the document",
  "document.unarchived": () => "unarchived the document",
  "document.deleted": () => "deleted the document",
  "sharing.granted": (a) => `shared with ${a.target.name} as ${a.role}`,
  "sharing.invitation_sent": (a) => `invited ${a.target.name} as ${a.role}`,
  "sharing.role_changed": (a) => `changed ${a.target.name} to ${a.role}`,
  "sharing.revoked": (a) => `removed ${a.target.name}`,
  "sharing.invitation_cancelled": (a) =>
    `cancelled the invitation of ${a.target.name}`,
  "sharing.invitation_accepted": (a) => `accepted an invitation as ${a.role}`,
  "sharing.ownership_transferred": (a) =>
    `transferred ownership to ${a.target.name}`,
  "log.created": (a) => `created log ${t(a)}`,
  "log.edited": (a) => `edited log ${t(a)}${fields(a)}`,
  // The example-logs bulk delete has no id; its name is "example logs".
  "log.deleted": (a) =>
    a.target.id ? `deleted log ${t(a)}` : `deleted ${a.target.name}`,
  "log.status_changed": (a) => `changed the status of log ${t(a)}`,
  "telegram.linked": () => "linked a Telegram account",
  "telegram.unlinked": () => "unlinked a Telegram account",
  "export.requested": () => "requested a PDF report",
};

/** "edited log “Migrated billing” (name, impact)": names and field names only (FR-13). */
export const describeAction = (a: AuditEntry) => verbs[a.action](a);

export const actorName = (a: AuditEntry) =>
  a.actor.name || a.actor.email || "System";

export const actionLabels = Object.keys(verbs) as AuditAction[];
