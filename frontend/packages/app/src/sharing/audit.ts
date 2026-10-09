import type { AuditEntry } from "../lib/types";

const verbs: Record<AuditEntry["action"], string> = {
  grant: "shared",
  invite: "invited",
  role_change: "changed the role of",
  revoke: "removed",
  invite_cancel: "cancelled the invitation of",
  invite_accept: "accepted an invitation as",
  transfer: "transferred ownership to",
};

/** "ada@x shared bob@x on “2026” as viewer" */
export function describeAudit(a: AuditEntry) {
  const as =
    a.action === "grant" || a.action === "invite" || a.action === "role_change"
      ? ` as ${a.role}`
      : "";
  return `${a.actor_email} ${verbs[a.action]} ${a.target} on “${a.document_title}”${as}`;
}
