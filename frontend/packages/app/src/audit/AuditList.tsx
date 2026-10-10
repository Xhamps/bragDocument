import type { ReactNode } from "react";
import { Button, Tag, cn } from "@bragdoc/ui";
import type { AuditEntry } from "../lib/types";
import { actorName, describeAction } from "./describe";
import { When } from "./When";

/**
 * Audit entries as a list in columns: user, action, (document), source, time.
 * With onActor/onDocument the names become filter buttons (the Audit log page).
 */
export function AuditList({
  entries,
  empty,
  showDocument = true,
  onActor,
  onDocument,
}: {
  entries: AuditEntry[];
  empty: ReactNode;
  showDocument?: boolean;
  onActor?: (id: string) => void;
  onDocument?: (id: string) => void;
}) {
  // One template for the header and every row, so the columns line up.
  const cols = cn(
    "grid gap-x-4 gap-y-1 md:items-center",
    showDocument
      ? "md:grid-cols-[minmax(0,10rem)_minmax(0,1fr)_minmax(0,8rem)_6rem_7rem]"
      : "md:grid-cols-[minmax(0,10rem)_minmax(0,1fr)_6rem_7rem]",
  );
  const filter = (label: string, onClick: () => void) => (
    <Button
      variant="tinted"
      size="sm"
      className="h-auto justify-start truncate px-0"
      onClick={onClick}
    >
      {label}
    </Button>
  );
  return (
    <div className="rounded-md border text-sm">
      <div
        aria-hidden
        className={cn(
          cols,
          "hidden border-b px-4 py-2 text-xs font-medium text-fg-secondary md:grid",
        )}
      >
        <span>User</span>
        <span>Action</span>
        {showDocument && <span>Document</span>}
        <span>Source</span>
        <span className="md:text-right">When</span>
      </div>
      <ul className="flex flex-col divide-y">
        {entries.length === 0 && (
          <li className="px-4 py-3 text-fg-secondary">{empty}</li>
        )}
        {entries.map((e) => {
          const actor = e.actor.id;
          const doc = e.document;
          return (
            <li key={e.id} className={cn(cols, "px-4 py-3")}>
              <span className="truncate font-medium">
                {actor && onActor
                  ? filter(actorName(e), () => onActor(actor))
                  : actorName(e)}
              </span>
              <span>{describeAction(e)}</span>
              {showDocument && (
                <span className="truncate text-fg-secondary">
                  {doc &&
                    (onDocument
                      ? filter(doc.title, () => onDocument(doc.id))
                      : doc.title)}
                </span>
              )}
              <span>
                <Tag variant="outline">{e.source}</Tag>
              </span>
              <span className="text-fg-secondary md:text-right">
                <When at={e.at} />
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
