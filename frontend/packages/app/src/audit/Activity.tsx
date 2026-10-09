import { Link } from "react-router";
import { Button } from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import { actorName, describeAction } from "./describe";
import { useDocumentActivity } from "./useAudit";
import { When } from "./When";

/** The document page's Activity (FR-11): owners only; "View all" opens the Audit log filtered. */
export function Activity({ docId }: { docId: string }) {
  const q = useDocumentActivity(docId, true);
  const entries = q.data?.pages[0]?.entries ?? [];
  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Activity</h3>
        <Link
          to={`/audit?document=${docId}`}
          className="text-sm text-muted-foreground hover:underline"
        >
          View all
        </Link>
      </div>
      {q.error ? (
        <p role="alert" className="text-sm text-destructive">
          {errorText(q.error)}{" "}
          <Button variant="link" size="sm" onClick={() => q.refetch()}>
            Retry
          </Button>
        </p>
      ) : (
        <ul className="flex flex-col gap-1 text-sm">
          {q.isPending && (
            <li className="h-4 animate-pulse rounded bg-muted" aria-hidden />
          )}
          {!q.isPending && entries.length === 0 && (
            <li className="text-muted-foreground">No activity yet.</li>
          )}
          {entries.map((e) => (
            <li key={e.id}>
              {actorName(e)} {describeAction(e)} ·{" "}
              <span className="text-muted-foreground">
                <When at={e.at} />
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
