import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import type { Log } from "../lib/types";
import { DocumentTabs } from "../documents/DocumentTabs";
import { useDocument } from "../documents/useDocuments";
import { ApiError } from "../lib/api";
import { SharePanel } from "../sharing/SharePanel";
import { LogFormDialog } from "../logs/LogFormDialog";
import { ActiveFilters, LogFilters } from "../logs/LogFilters";
import { LogRow } from "../logs/LogRow";
import {
  useCreateLog,
  useDeleteExamples,
  useDeleteLog,
  useLog,
  useLogs,
  useUpdateLog,
  type LogForm,
} from "../logs/useLogs";

const PER_PAGE = 50; // the API default; the URL may override it with per_page

export function Component() {
  const { id = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const docQuery = useDocument(id);
  const [sharing, setSharing] = useState(false);
  // The bot's deep link (PRD-0003 FR-6): `edit` is a UI param, not a filter.
  const editId = params.get("edit");
  const listParams = new URLSearchParams(params);
  listParams.delete("edit");
  const logs = useLogs(id, listParams.toString());
  const deepLinked = useLog(id, editId);
  const create = useCreateLog(id);
  const update = useUpdateLog(id);
  const remove = useDeleteLog(id);
  const removeExamples = useDeleteExamples(id);
  const [editing, setEditing] = useState<Log | "new" | null>(null);
  const [noImpact, setNoImpact] = useState(false);
  const [deleting, setDeletingLog] = useState<Log | null>(null);
  const setDeleting = (log: Log | null) => {
    setDeletingLog(log);
    remove.reset(); // don't carry a failed delete's error to the next dialog
  };
  // Drop the deep-link param once the fetch settles; opening is below.
  useEffect(() => {
    if (!editId || deepLinked.isPending) return;
    setParams(
      (p) => {
        const n = new URLSearchParams(p);
        n.delete("edit");
        return n;
      },
      { replace: true },
    );
  }, [editId, deepLinked.isPending, setParams]);

  if (docQuery.isPending)
    return <p className="text-muted-foreground">Loading…</p>;
  const doc = docQuery.data;
  if (!doc)
    return (
      <p role="alert" className="text-destructive">
        {docQuery.error instanceof ApiError && docQuery.error.status === 404
          ? "Document not found."
          : errorText(docQuery.error)}
      </p>
    );
  const archived = doc.state === "archived";
  const readOnly = archived || doc.role === "viewer";
  // Open the deep-linked log (adjusting state during render); a missing log or
  // a read-only (archived or viewer) document is ignored.
  if (editId && deepLinked.data && editing === null && !readOnly)
    setEditing(deepLinked.data);

  const setFilter = (key: string, values: string[]) => {
    const next = new URLSearchParams(params);
    next.delete(key);
    values.forEach((v) => next.append(key, v));
    next.delete("page");
    setParams(next);
  };
  const setPage = (p: number) => {
    const next = new URLSearchParams(params);
    if (p > 1) next.set("page", String(p));
    else next.delete("page");
    setParams(next);
  };
  const clearFilters = () => {
    const sort = params.get("sort");
    setParams(sort ? { sort } : {});
  };

  const closeForm = () => {
    setEditing(null);
    setNoImpact(false);
    create.reset();
    update.reset();
  };
  // PRD-0007 FR-4: no impact found → keep the form open on the saved log.
  const saved = (log: Log) => {
    if (log.impact_statement === "") {
      setEditing(log);
      setNoImpact(true);
    } else closeForm();
  };
  const submit = (form: LogForm) => {
    if (editing === "new") create.mutate(form, { onSuccess: saved });
    else if (editing) {
      // The backend re-extracts only when name or description changed.
      const extracted =
        form.name !== editing.name || form.description !== editing.description;
      update.mutate(
        { id: editing.id, ...form },
        { onSuccess: extracted ? saved : closeForm },
      );
    }
  };

  const page = Math.max(1, Number(params.get("page")) || 1);
  const perPage = Math.min(
    100,
    Math.max(1, Number(params.get("per_page")) || PER_PAGE),
  );
  const total = logs.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / perPage));
  const filtered = [...listParams.keys()].some(
    (k) => k !== "sort" && k !== "page" && k !== "per_page",
  );
  const items = logs.data?.items ?? [];

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-4">
        <Link to="/" className="text-sm text-muted-foreground hover:underline">
          ← Documents
        </Link>
        <h2 className="text-xl font-semibold">{doc.title}</h2>
        <DocumentTabs id={doc.id} current="logs" />
        {doc.role !== "owner" && (
          <span className="text-sm text-muted-foreground">
            {`Shared by ${doc.owner_name} · you are ${doc.role}`}
          </span>
        )}
        <div className="ml-auto flex items-center gap-2">
          {archived ? (
            <Badge variant="secondary">Archived · read-only</Badge>
          ) : doc.role === "viewer" ? (
            <Badge variant="secondary">Viewer · read-only</Badge>
          ) : (
            <Button onClick={() => setEditing("new")}>New log</Button>
          )}
          {doc.role === "owner" && (
            <Button variant="outline" onClick={() => setSharing(true)}>
              Share
            </Button>
          )}
        </div>
      </div>

      <LogFilters params={params} onChange={setFilter} />
      <ActiveFilters
        params={params}
        onRemove={(k, v) =>
          setFilter(
            k,
            params.getAll(k).filter((x) => x !== v),
          )
        }
        onClear={clearFilters}
      />

      {logs.error ? (
        <p role="alert" className="text-destructive">
          {errorText(logs.error)}
        </p>
      ) : logs.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : (
        <>
          <p aria-live="polite" className="text-sm text-muted-foreground">
            {total} {total === 1 ? "log" : "logs"}
          </p>
          {!readOnly && items.some((l) => l.is_example) && (
            <div className="flex flex-wrap items-center gap-3 rounded-md border border-dashed p-3 text-sm">
              <span>
                Example logs show what a good entry looks like: what you did,
                the result, and the evidence.
              </span>
              <Button
                size="sm"
                variant="outline"
                disabled={removeExamples.isPending}
                onClick={() => removeExamples.mutate(undefined)}
              >
                Remove examples
              </Button>
              {removeExamples.error && (
                <p role="alert" className="text-destructive">
                  {errorText(removeExamples.error)}
                </p>
              )}
            </div>
          )}
          {total === 0 ? (
            filtered ? (
              <p className="text-muted-foreground">
                No logs match these filters.{" "}
                <Button variant="link" onClick={clearFilters}>
                  Clear filters
                </Button>
              </p>
            ) : (
              <p className="text-muted-foreground">
                No logs yet. Capture your latest win while you still remember
                the details.
              </p>
            )
          ) : items.length === 0 ? (
            <p className="text-muted-foreground">
              This page is empty.{" "}
              <Button variant="link" onClick={() => setPage(1)}>
                Go to page 1
              </Button>
            </p>
          ) : (
            <ul className="flex flex-col divide-y rounded-md border">
              {items.map((l) => (
                <LogRow
                  key={l.id}
                  log={l}
                  readOnly={readOnly}
                  onEdit={setEditing}
                  onDelete={setDeleting}
                />
              ))}
            </ul>
          )}
          {pages > 1 && (
            <nav aria-label="Pagination" className="flex items-center gap-3">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                Previous
              </Button>
              <span className="text-sm">
                Page {page} of {pages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= pages}
                onClick={() => setPage(page + 1)}
              >
                Next
              </Button>
            </nav>
          )}
        </>
      )}

      <LogFormDialog
        open={editing !== null}
        formKey={editing === "new" ? "new" : (editing?.id ?? "none")}
        onOpenChange={(o) => !o && closeForm()}
        initial={editing === "new" ? undefined : (editing ?? undefined)}
        busy={create.isPending || update.isPending}
        error={errorText(create.error ?? update.error)}
        noImpact={noImpact}
        onSubmit={submit}
      />

      <Dialog
        open={deleting !== null}
        onOpenChange={(o) => !o && setDeleting(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete log?</DialogTitle>
            <DialogDescription>
              “{deleting?.name}” will be removed permanently.
            </DialogDescription>
          </DialogHeader>
          {remove.error && (
            <p role="alert" className="text-sm text-destructive">
              {errorText(remove.error)}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleting(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={remove.isPending}
              onClick={() =>
                deleting &&
                remove.mutate(deleting.id, {
                  onSuccess: () => setDeleting(null),
                })
              }
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SharePanel doc={doc} open={sharing} onOpenChange={setSharing} />
    </div>
  );
}
