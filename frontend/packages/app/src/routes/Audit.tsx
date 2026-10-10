import { useSearchParams } from "react-router";
import {
  Button,
  DataTable,
  Select,
  TextField,
  type DataTableColumn,
} from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import { actionLabels, actorName, describeAction } from "../audit/describe";
import { useAudit, useAuditFilters, type AuditQuery } from "../audit/useAudit";
import { When } from "../audit/When";
import type { AuditEntry } from "../lib/types";

const keys = ["actor", "document", "action", "from", "to"] as const;
type Key = (typeof keys)[number];

/** Filters live in the URL so a filtered view can be shared (PRD-0009 §9). */
export function Component() {
  const [params, setParams] = useSearchParams();
  const filters = Object.fromEntries(
    keys.map((k) => [k, params.get(k) ?? ""]),
  ) as Required<AuditQuery>;
  const filtered = keys.some((k) => filters[k]);
  const audit = useAudit(filters);
  const options = useAuditFilters();

  const set = (k: Key, v: string) =>
    setParams((p) => {
      const next = new URLSearchParams(p);
      if (v) next.set(k, v);
      else next.delete(k);
      return next;
    });

  const entries = audit.data?.pages.flatMap((p) => p.entries) ?? [];
  const link = (label: string, onClick: () => void) => (
    <Button variant="tinted" size="sm" className="px-0" onClick={onClick}>
      {label}
    </Button>
  );
  const columns: DataTableColumn<AuditEntry>[] = [
    {
      key: "at",
      header: "Time",
      render: (e) => <When at={e.at} />,
    },
    {
      key: "actor",
      header: "User",
      render: (e) =>
        e.actor.id
          ? link(actorName(e), () => set("actor", e.actor.id!))
          : actorName(e),
    },
    { key: "action", header: "Action", wrap: true, render: describeAction },
    {
      key: "document",
      header: "Target",
      render: (e) =>
        e.document &&
        link(e.document.title, () => set("document", e.document!.id)),
    },
    { key: "source", header: "Source" },
  ];

  return (
    <div className="flex flex-col gap-4">
      <h2 className="type-title-2">Audit log</h2>
      <div className="flex flex-wrap gap-2">
        <Select
          aria-label="User"
          value={filters.actor}
          onChange={(e) => set("actor", e.target.value)}
          options={[
            { value: "", label: "All users" },
            ...(options.data?.actors ?? []).map((a) => ({
              value: a.id,
              label: a.name || a.email,
            })),
          ]}
        />
        <Select
          aria-label="Document"
          value={filters.document}
          onChange={(e) => set("document", e.target.value)}
          options={[
            { value: "", label: "All documents" },
            ...(options.data?.documents ?? []).map((d) => ({
              value: d.id,
              label: d.title,
            })),
          ]}
        />
        <Select
          aria-label="Action"
          value={filters.action}
          onChange={(e) => set("action", e.target.value)}
          options={[{ value: "", label: "All actions" }, ...actionLabels]}
        />
        <TextField
          type="date"
          aria-label="From"
          className="w-40"
          value={filters.from}
          max={filters.to || undefined}
          onChange={(e) => set("from", e.target.value)}
        />
        <TextField
          type="date"
          aria-label="To"
          className="w-40"
          value={filters.to}
          min={filters.from || undefined}
          onChange={(e) => set("to", e.target.value)}
        />
      </div>

      {audit.error ? (
        <p role="alert" className="text-sm text-danger">
          {errorText(audit.error)}{" "}
          <Button variant="tinted" size="sm" onClick={() => audit.refetch()}>
            Retry
          </Button>
        </p>
      ) : (
        <DataTable
          caption="Audit log"
          density="compact"
          columns={columns}
          rows={entries}
          empty={
            audit.isPending ? (
              "Loading…"
            ) : filtered ? (
              <>
                No entries match these filters{" "}
                <Button
                  variant="tinted"
                  size="sm"
                  onClick={() => setParams({})}
                >
                  Clear filters
                </Button>
              </>
            ) : (
              "No activity yet."
            )
          }
          footer={
            audit.hasNextPage && (
              <Button
                variant="glass"
                disabled={audit.isFetchingNextPage}
                onClick={() => audit.fetchNextPage()}
              >
                Load more
              </Button>
            )
          }
        />
      )}
    </div>
  );
}
