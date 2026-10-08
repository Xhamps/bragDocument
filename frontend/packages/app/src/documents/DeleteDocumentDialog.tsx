import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@bragdoc/ui";
import type { Document } from "../lib/types";

type Props = {
  doc: Document | null;
  busy?: boolean;
  error?: string | null;
  onCancel: () => void;
  onConfirm: (doc: Document) => void;
};

export function DeleteDocumentDialog({
  doc,
  busy,
  error,
  onCancel,
  onConfirm,
}: Props) {
  return (
    <Dialog open={doc !== null} onOpenChange={(o) => !o && onCancel()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete "{doc?.title}"?</DialogTitle>
          <DialogDescription>
            All its logs and sharing grants are deleted too. This cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={busy}
            onClick={() => doc && onConfirm(doc)}
          >
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
