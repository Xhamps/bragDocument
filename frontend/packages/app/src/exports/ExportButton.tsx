import { useState } from "react";
import { Button } from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import type { DocRole } from "../lib/types";
import { ExportDialog } from "./ExportDialog";
import { downloadJob, useExportJob } from "./useExports";

/** "Export PDF" for the document header (FR-1) plus a live status of the last export. */
export function ExportButton({
  docId,
  role,
  params,
}: {
  docId: string;
  role: DocRole;
  params: URLSearchParams;
}) {
  const [open, setOpen] = useState(false);
  const [jobId, setJobId] = useState<string | null>(null);
  const [dlError, setDlError] = useState<unknown>(null);
  const { data: job, error } = useExportJob(docId, jobId);
  return (
    <>
      <Button variant="glass" onClick={() => setOpen(true)}>
        Export PDF
      </Button>
      <ExportDialog
        docId={docId}
        role={role}
        params={params}
        open={open}
        onOpenChange={setOpen}
        onStarted={(j) => setJobId(j.id)}
      />
      {/* Always mounted so screen readers announce changes. */}
      <span role="status" className="flex items-center gap-2 text-sm">
        {error ? (
          <span className="text-destructive">{errorText(error)}</span>
        ) : !job ? null : job.status === "done" ? (
          <>
            Report ready
            <Button
              size="sm"
              variant="tinted"
              onClick={() =>
                downloadJob(docId, job).then(() => setDlError(null), setDlError)
              }
            >
              Download
            </Button>
            {!!dlError && (
              <span className="text-destructive">{errorText(dlError)}</span>
            )}
          </>
        ) : job.status === "failed" ? (
          <span className="text-destructive">
            {job.error || "Export failed"}
          </span>
        ) : (
          <>
            Generating report
            <progress
              max={100}
              value={job.progress}
              aria-label="Report progress"
            />
          </>
        )}
      </span>
    </>
  );
}
