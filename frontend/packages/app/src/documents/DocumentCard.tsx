import {
  Badge,
  Button,
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
import { MoreHorizontalIcon } from "lucide-react";
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
        <CardTitle>{doc.title}</CardTitle>
        {doc.description && (
          <CardDescription>{doc.description}</CardDescription>
        )}
        {!readOnly && (
          <CardAction>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon-sm" aria-label="Actions">
                  <MoreHorizontalIcon />
                </Button>
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
                  variant="destructive"
                  onSelect={() => onDelete(doc)}
                >
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </CardAction>
        )}
      </CardHeader>
      <CardContent className="flex items-center gap-2 text-muted-foreground">
        <span>Updated {new Date(doc.updated_at).toLocaleDateString()}</span>
        {archived && <Badge variant="secondary">Archived</Badge>}
      </CardContent>
    </Card>
  );
}
