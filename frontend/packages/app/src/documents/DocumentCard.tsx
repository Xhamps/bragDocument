import {
  Badge,
  IconButton,
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
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
    <Card className={archived ? "opacity-60" : undefined}>
      <CardHeader>
        <CardTitle>
          <Link to={`/documents/${doc.id}`} className="hover:underline">
            {doc.title}
          </Link>
        </CardTitle>
        {doc.description && (
          <CardDescription>{doc.description}</CardDescription>
        )}
        {!readOnly && (
          <CardAction>
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
          </CardAction>
        )}
      </CardHeader>
      <CardContent className="text-muted-foreground flex flex-wrap items-center gap-2">
        <span>
          {doc.log_count} {doc.log_count === 1 ? "log" : "logs"}
          {doc.last_log_at &&
            ` · last on ${new Date(doc.last_log_at).toLocaleDateString()}`}
          {" ·"}
        </span>
        <span>Updated {new Date(doc.updated_at).toLocaleDateString()}</span>
        {archived && <Badge variant="secondary">Archived</Badge>}
        {doc.role !== "owner" && (
          <>
            <span>Shared by {doc.owner_name}</span>
            <Badge variant="outline">{doc.role}</Badge>
            {doc.is_new && <Badge>New</Badge>}
          </>
        )}
      </CardContent>
    </Card>
  );
}
