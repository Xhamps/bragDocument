import { describeAction, actorName } from "./describe";
import type { AuditEntry } from "../lib/types";

const base: AuditEntry = {
  id: 1,
  at: "2026-10-09T12:00:00Z",
  source: "web",
  action: "log.edited",
  actor: { id: "u1", name: "Ana", email: "ana@acme.com" },
  document: { id: "d1", title: "2026" },
  target: { type: "log", id: "l1", name: "Migrated billing" },
  role: "",
  changed_fields: ["name", "impact"],
};

test("describes actions without content", () => {
  expect(describeAction(base)).toBe(
    "edited log “Migrated billing” (name, impact)",
  );
  expect(
    describeAction({
      ...base,
      action: "log.status_changed",
      changed_fields: ["status"],
    }),
  ).toBe("changed the status of log “Migrated billing”");
  expect(
    describeAction({
      ...base,
      action: "sharing.granted",
      target: { type: "user", id: "u2", name: "bob@acme.com" },
      role: "viewer",
    }),
  ).toBe("shared with bob@acme.com as viewer");
  expect(
    describeAction({
      ...base,
      action: "telegram.linked",
      document: null,
      target: { type: "", id: "", name: "" },
    }),
  ).toBe("linked a Telegram account");
  expect(
    describeAction({
      ...base,
      action: "log.deleted",
      target: { type: "log", id: "", name: "example logs" },
      changed_fields: [],
    }),
  ).toBe("deleted example logs");
});

test("actor falls back to email, then System", () => {
  expect(actorName(base)).toBe("Ana");
  expect(
    actorName({
      ...base,
      actor: { id: "u1", name: "", email: "ana@acme.com" },
    }),
  ).toBe("ana@acme.com");
  expect(actorName({ ...base, actor: { id: null, name: "", email: "" } })).toBe(
    "System",
  );
});
