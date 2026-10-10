import {
  IconButton,
  Card,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Tag,
  focusRing,
} from "@bragdoc/ui";
import { Link } from "react-router";
import type { Document } from "../lib/types";

type Props = {
  doc: Document;
  readOnly?: boolean;
  onRename: (doc: Document) => void;
  onToggleArchive: (doc: Document) => void;
  onDelete: (doc: Document) => void;
};

export function DocumentCard({
  doc,
  readOnly,
  onRename,
  onToggleArchive,
  onDelete,
}: Props) {
  const archived = doc.state === "archived";
  return (
    <Card
      interactive
      className={archived ? "opacity-60" : undefined}
      title={
        // Stretched link: the whole card navigates; the menu sits above it.
        <Link
          to={`/documents/${doc.id}`}
          className={`${focusRing} rounded-sm before:absolute before:inset-0 hover:underline`}
        >
          {doc.title}
        </Link>
      }
      description={doc.description || undefined}
      action={
        !readOnly && (
          <div className="relative z-10">
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
                <DropdownMenuItem onSelect={() => onRename(doc)}>
                  Rename
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => onToggleArchive(doc)}>
                  {archived ? "Unarchive" : "Archive"}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="danger"
                  onSelect={() => onDelete(doc)}
                >
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )
      }
    >
      <div className="mt-4 flex flex-wrap items-center gap-2 type-footnote text-fg-secondary">
        <span>
          {doc.log_count} {doc.log_count === 1 ? "log" : "logs"}
          {doc.last_log_at &&
            ` · last on ${new Date(doc.last_log_at).toLocaleDateString()}`}
          {" ·"}
        </span>
        <span>Updated {new Date(doc.updated_at).toLocaleDateString()}</span>
        {archived && <Tag>Archived</Tag>}
        {doc.role !== "owner" && (
          <>
            <span>Shared by {doc.owner_name}</span>
            <Tag
              variant="outline"
              tone={doc.role === "editor" ? "accent" : "neutral"}
            >
              {doc.role}
            </Tag>
            {doc.is_new && <Tag tone="accent">New</Tag>}
          </>
        )}
      </div>
    </Card>
  );
}
