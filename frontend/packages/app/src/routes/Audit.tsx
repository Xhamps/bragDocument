import { useSearchParams } from "react-router";
import { Button, TextField } from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import { FIELD } from "../logs/constants";
import { actionLabels, actorName, describeAction } from "../audit/describe";
import { useAudit, useAuditFilters, type AuditQuery } from "../audit/useAudit";
import { When } from "../audit/When";

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

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-xl font-semibold">Audit log</h2>
      <div className="flex flex-wrap gap-2">
        <select
          aria-label="User"
          className={FIELD}
          value={filters.actor}
          onChange={(e) => set("actor", e.target.value)}
        >
          <option value="">All users</option>
          {options.data?.actors.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name || a.email}
            </option>
          ))}
        </select>
        <select
          aria-label="Document"
          className={FIELD}
          value={filters.document}
          onChange={(e) => set("document", e.target.value)}
        >
          <option value="">All documents</option>
          {options.data?.documents.map((d) => (
            <option key={d.id} value={d.id}>
              {d.title}
            </option>
          ))}
        </select>
        <select
          aria-label="Action"
          className={FIELD}
          value={filters.action}
          onChange={(e) => set("action", e.target.value)}
        >
          <option value="">All actions</option>
          {actionLabels.map((a) => (
            <option key={a} value={a}>
              {a}
            </option>
          ))}
        </select>
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
        <p role="alert" className="text-destructive text-sm">
          {errorText(audit.error)}{" "}
          <Button variant="tinted" size="sm" onClick={() => audit.refetch()}>
            Retry
          </Button>
        </p>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-muted-foreground text-left">
            <tr>
              <th className="py-2 font-medium">Time</th>
              <th className="font-medium">User</th>
              <th className="font-medium">Action</th>
              <th className="font-medium">Target</th>
              <th className="font-medium">Source</th>
            </tr>
          </thead>
          <tbody>
            {audit.isPending &&
              [0, 1, 2].map((i) => (
                <tr key={i} aria-hidden>
                  <td colSpan={5}>
                    <div className="bg-muted my-2 h-4 animate-pulse rounded" />
                  </td>
                </tr>
              ))}
            {entries.map((e) => (
              <tr key={e.id} className="border-t">
                <td className="py-2 whitespace-nowrap">
                  <When at={e.at} />
                </td>
                <td>
                  {e.actor.id ? (
                    <Button
                      variant="tinted"
                      size="sm"
                      className="px-0"
                      onClick={() => set("actor", e.actor.id!)}
                    >
                      {actorName(e)}
                    </Button>
                  ) : (
                    actorName(e)
                  )}
                </td>
                <td>{describeAction(e)}</td>
                <td>
                  {e.document && (
                    <Button
                      variant="tinted"
                      size="sm"
                      className="px-0"
                      onClick={() => set("document", e.document!.id)}
                    >
                      {e.document.title}
                    </Button>
                  )}
                </td>
                <td>{e.source}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {!audit.isPending &&
        !audit.error &&
        entries.length === 0 &&
        (filtered ? (
          <p className="text-muted-foreground text-sm">
            No entries match these filters{" "}
            <Button variant="tinted" size="sm" onClick={() => setParams({})}>
              Clear filters
            </Button>
          </p>
        ) : (
          <p className="text-muted-foreground text-sm">No activity yet.</p>
        ))}

      {audit.hasNextPage && (
        <Button
          variant="glass"
          className="self-start"
          disabled={audit.isFetchingNextPage}
          onClick={() => audit.fetchNextPage()}
        >
          Load more
        </Button>
      )}
    </div>
  );
}
