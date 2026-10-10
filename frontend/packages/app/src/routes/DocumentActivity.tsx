import { Link, Navigate, useParams } from "react-router";
import { Button } from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import { AuditView } from "../audit/AuditView";
import { useAuditParams, useDocumentActivity } from "../audit/useAudit";
import { useMe } from "../auth/useMe";
import { DocumentPage } from "../documents/DocumentPage";
import { useDocument } from "../documents/useDocuments";

/** The document's Activity tab (PRD-0009 FR-11); "View all" opens the Audit log filtered. */
export function Component() {
  const { id = "" } = useParams();
  const doc = useDocument(id);
  const { data: me } = useMe();
  // Same rule as the tab in DocumentPage.
  const allowed =
    doc.data?.role === "owner" || (!!doc.data && me?.role === "admin");
  const url = useAuditParams();
  const q = useDocumentActivity(id, url.filters, allowed);

  // "View all" carries this tab's filters over to the Audit log.
  const nonEmpty = Object.fromEntries(
    Object.entries(url.filters).filter(([, v]) => v),
  );

  if (doc.isPending || !me)
    return <p className="text-fg-secondary">Loading…</p>;
  if (!doc.data)
    return (
      <p role="alert" className="text-danger">
        {doc.error instanceof ApiError && doc.error.status === 404
          ? "Document not found."
          : errorText(doc.error)}
      </p>
    );
  if (!allowed) return <Navigate to={`/documents/${id}`} replace />;

  return (
    <div className="flex flex-col gap-6">
      <DocumentPage
        doc={doc.data}
        current="activity"
        actions={
          <Button variant="glass" asChild>
            <Link
              to={`/audit?${new URLSearchParams({ ...nonEmpty, document: id })}`}
            >
              View all in Audit log
            </Link>
          </Button>
        }
      >
        <AuditView query={q} url={url} byDocument={false} />
      </DocumentPage>
    </div>
  );
}
