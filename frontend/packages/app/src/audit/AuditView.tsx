import { Button, Select, TextField } from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import { AuditList } from "./AuditList";
import { actionLabels } from "./describe";
import {
  useAudit,
  useAuditFilters,
  type AuditKey,
  type AuditParams,
} from "./useAudit";

/**
 * Filters over the audit list, with Retry and Load more. The Audit log page
 * filters by document too; a document's Activity tab is already one document.
 */
export function AuditView({
  query,
  url: { filters, set, clear },
  byDocument = true,
}: {
  query: ReturnType<typeof useAudit>;
  url: AuditParams;
  byDocument?: boolean;
}) {
  const options = useAuditFilters();
  const used: AuditKey[] = ["actor", "action", "from", "to"];
  if (byDocument) used.push("document");
  const filtered = used.some((k) => filters[k]);
  const entries = query.data?.pages.flatMap((p) => p.entries) ?? [];

  return (
    <>
      <div
        className={
          byDocument
            ? "grid gap-2 sm:grid-cols-2 lg:grid-cols-5"
            : "grid gap-2 sm:grid-cols-2 lg:grid-cols-4"
        }
      >
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
        {byDocument && (
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
        )}
        <Select
          aria-label="Action"
          value={filters.action}
          onChange={(e) => set("action", e.target.value)}
          options={[{ value: "", label: "All actions" }, ...actionLabels]}
        />
        <TextField
          type="date"
          aria-label="From"
          value={filters.from}
          max={filters.to || undefined}
          onChange={(e) => set("from", e.target.value)}
        />
        <TextField
          type="date"
          aria-label="To"
          value={filters.to}
          min={filters.from || undefined}
          onChange={(e) => set("to", e.target.value)}
        />
      </div>

      {query.error ? (
        <p role="alert" className="text-sm text-danger">
          {errorText(query.error)}{" "}
          <Button variant="tinted" size="sm" onClick={() => query.refetch()}>
            Retry
          </Button>
        </p>
      ) : (
        <>
          <AuditList
            entries={entries}
            showDocument={byDocument}
            onActor={(id) => set("actor", id)}
            onDocument={byDocument ? (id) => set("document", id) : undefined}
            empty={
              query.isPending ? (
                "Loading…"
              ) : filtered ? (
                <>
                  No entries match these filters{" "}
                  <Button variant="tinted" size="sm" onClick={clear}>
                    Clear filters
                  </Button>
                </>
              ) : (
                "No activity yet."
              )
            }
          />
          {query.hasNextPage && (
            <Button
              variant="glass"
              className="self-center"
              disabled={query.isFetchingNextPage}
              onClick={() => query.fetchNextPage()}
            >
              Load more
            </Button>
          )}
        </>
      )}
    </>
  );
}
