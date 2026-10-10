import { useState } from "react";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Label,
} from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { DocRole, ExportJob, ReportSettings } from "../lib/types";
import { FIELD } from "../logs/constants";
import { useTags } from "../logs/useLogs";
import { SECTIONS } from "./sections";
import {
  downloadJob,
  useCreateExport,
  useExportHistory,
  useReportSettings,
} from "./useExports";

/** Filters only: paging, sorting, and UI params do not change the report. */
function exportQuery(params: URLSearchParams) {
  const q = new URLSearchParams(params);
  // Must match the UI-only params of DocumentLogs and Dashboard.
  for (const k of ["page", "per_page", "sort", "period", "edit"]) q.delete(k);
  return q.toString();
}

export function ExportDialog({
  docId,
  role,
  params,
  open,
  onOpenChange,
  onStarted,
}: {
  docId: string;
  role: DocRole;
  params: URLSearchParams;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onStarted: (j: ExportJob) => void;
}) {
  const settings = useReportSettings(docId, open);
  const history = useExportHistory(docId, open);
  const tags = useTags();
  const create = useCreateExport(docId);
  const query = exportQuery(params);
  const [dlError, setDlError] = useState<unknown>(null);
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        create.reset(); // don't carry a failed export's error to the next open
      }}
    >
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Export PDF</DialogTitle>
          <DialogDescription>
            {query
              ? `Filters: ${[...new URLSearchParams(query)].map(([k, v]) => `${k}=${v}`).join(" · ")}`
              : "All logs"}
            . Example logs are left out.
          </DialogDescription>
        </DialogHeader>
        {settings.error && (
          <p role="alert" className="text-destructive text-sm">
            {errorText(settings.error)}
          </p>
        )}
        {settings.data && (
          <SettingsForm
            initial={settings.data}
            tags={tags.data ?? []}
            canSave={role !== "viewer"}
            pending={create.isPending}
            error={
              (create.error instanceof ApiError &&
                create.error.fields?.filters) ||
              errorText(create.error) ||
              undefined
            }
            onSubmit={(s) =>
              create.mutate(
                { ...s, query },
                {
                  onSuccess: (j) => {
                    onStarted(j);
                    onOpenChange(false);
                  },
                },
              )
            }
          />
        )}
        {!!history.data?.length && (
          <section
            aria-label="Recent exports"
            className="flex flex-col gap-1 text-sm"
          >
            <h3 className="font-medium">Recent exports</h3>
            {history.data.map((j) => (
              <div key={j.id} className="flex items-center gap-2">
                <span>{new Date(j.created_at).toLocaleString()}</span>
                <span className="text-muted-foreground">
                  {j.status === "failed" ? j.error : j.status}
                </span>
                {j.downloadable && (
                  <Button
                    size="sm"
                    variant="tinted"
                    onClick={() =>
                      downloadJob(docId, j).then(
                        () => setDlError(null),
                        setDlError,
                      )
                    }
                  >
                    Download
                  </Button>
                )}
              </div>
            ))}
            {!!dlError && (
              <p role="alert" className="text-destructive">
                {errorText(dlError)}
              </p>
            )}
          </section>
        )}
      </DialogContent>
    </Dialog>
  );
}

function SettingsForm({
  initial,
  tags,
  canSave,
  pending,
  error,
  onSubmit,
}: {
  initial: ReportSettings;
  tags: string[];
  /** Viewers can export, but the API saves settings only for owners and editors. */
  canSave: boolean;
  pending: boolean;
  error?: string;
  onSubmit: (s: ReportSettings) => void;
}) {
  const [goalsThis, setGoalsThis] = useState(initial.goals_this_year);
  const [goalsNext, setGoalsNext] = useState(initial.goals_next_year);
  const [map, setMap] = useState(initial.section_map);
  const rows = [...new Set([...Object.keys(map), ...tags])].sort();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({
          goals_this_year: goalsThis,
          goals_next_year: goalsNext,
          section_map: map,
        });
      }}
    >
      <div className="flex flex-col gap-1">
        <Label htmlFor="goals-this-year">Goals for this year</Label>
        <textarea
          id="goals-this-year"
          className={`${FIELD} min-h-20 w-full`}
          value={goalsThis}
          onChange={(e) => setGoalsThis(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1">
        <Label htmlFor="goals-next-year">Goals for next year</Label>
        <textarea
          id="goals-next-year"
          className={`${FIELD} min-h-20 w-full`}
          value={goalsNext}
          onChange={(e) => setGoalsNext(e.target.value)}
        />
      </div>
      {!canSave && (
        <p className="text-muted-foreground text-sm">
          These apply to this report only.
        </p>
      )}
      {rows.length > 0 && (
        <table className="text-sm">
          <caption className="text-left font-medium">
            Sections by tag (untagged and unmapped logs go to Other)
          </caption>
          <tbody>
            {rows.map((t) => (
              <tr key={t}>
                <th scope="row" className="py-1 pr-4 text-left font-normal">
                  {t}
                </th>
                <td>
                  <select
                    aria-label={`Section for ${t}`}
                    className={FIELD}
                    value={map[t] ?? "Other"}
                    onChange={(e) => setMap({ ...map, [t]: e.target.value })}
                  >
                    {SECTIONS.map((s) => (
                      <option key={s}>{s}</option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {error && (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      )}
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={pending}>
          {pending ? "Starting…" : "Generate"}
        </Button>
      </DialogFooter>
    </form>
  );
}
