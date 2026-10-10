import { useState, type ComponentProps } from "react";
import Markdown from "react-markdown";
import { TriangleAlertIcon } from "lucide-react";
import {
  Badge,
  IconButton,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@bragdoc/ui";
import type { Log } from "../lib/types";
import { STATUS_LABEL } from "./constants";

/**
 * Evidence links open elsewhere and never leak the referrer (PRD-0002 NFR-3).
 * Picks props explicitly: react-markdown also passes its hast `node`, which must not reach the DOM.
 */
export function ExternalLink({ href, title, children }: ComponentProps<"a">) {
  return (
    <a
      href={href}
      title={title}
      target="_blank"
      rel="noopener noreferrer"
      className="underline"
    >
      {children}
    </a>
  );
}

type Props = {
  log: Log;
  readOnly: boolean;
  onEdit: (log: Log) => void;
  onDelete: (log: Log) => void;
};

export function LogRow({ log, readOnly, onEdit, onDelete }: Props) {
  const [open, setOpen] = useState(false);
  return (
    <li className="flex flex-col gap-2 p-3">
      <div className="flex flex-wrap items-start gap-2">
        <button
          type="button"
          aria-expanded={open}
          onClick={() => setOpen(!open)}
          className="flex min-w-48 flex-1 flex-col items-start text-left"
        >
          <span className="font-medium">{log.name}</span>
          {log.impact_statement ? (
            <span className="text-muted-foreground text-sm">
              {log.impact_statement}
            </span>
          ) : log.impact_statement === "" ? (
            <span className="flex items-center gap-1 text-sm text-amber-700 dark:text-amber-400">
              <TriangleAlertIcon aria-hidden className="size-3.5" />
              No impact stated
            </span>
          ) : null}
        </button>
        {log.is_example && <Badge variant="secondary">Example</Badge>}
        <Badge
          variant={
            log.impact === "high" || log.impact === "critical"
              ? "default"
              : "secondary"
          }
        >
          {log.impact}
        </Badge>
        <Badge variant="outline">{STATUS_LABEL[log.status]}</Badge>
        <time
          dateTime={log.created_at}
          className="text-muted-foreground text-sm"
        >
          {new Date(log.created_at).toLocaleDateString()}
        </time>
        {!readOnly && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <IconButton
                variant="tinted"
                size="sm"
                icon="more"
                label="Actions"
                tooltip={false}
              />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => onEdit(log)}>
                Edit
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="danger" onSelect={() => onDelete(log)}>
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      {log.tags.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {log.tags.map((t) => (
            <Badge key={t} variant="outline">
              #{t}
            </Badge>
          ))}
        </div>
      )}
      {open && (
        <div className="flex flex-col gap-2 text-sm [&_ol]:list-decimal [&_ol]:pl-5 [&_ul]:list-disc [&_ul]:pl-5">
          {log.description && (
            <Markdown
              components={{ a: ExternalLink }}
              disallowedElements={["img"]}
              unwrapDisallowed
            >
              {log.description}
            </Markdown>
          )}
          {log.links.length > 0 && (
            <ul>
              {log.links.map((k, i) => (
                <li key={i}>
                  <ExternalLink href={k.url}>{k.label || k.url}</ExternalLink>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </li>
  );
}
